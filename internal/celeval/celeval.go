// Package celeval evaluates Tekton-Triggers-style CEL expressions against
// a raw HTTP request. The exposed bindings match Tekton:
//
//	body         dyn (parsed JSON body)
//	header       map<string, list<string>>  (lower-cased keys)
//	requestURL   string
//	method       string
//
// Plus member functions:
//
//	header.match(key, value)     -> bool
//	header.canonical(key)        -> string
package celeval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

type Request struct {
	CelExpression string `json:"celExpression"`
	HttpRequest   string `json:"httpRequest"`
}

type ParsedInput struct {
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Header     map[string][]string `json:"header"`
	Body       interface{}         `json:"body"`
	RawBody    string              `json:"rawBody"`
	BodyParsed bool                `json:"bodyParsed"`
}

type Response struct {
	Result      interface{}  `json:"result"`
	Error       string       `json:"error,omitempty"`
	ParsedInput *ParsedInput `json:"parsedInput,omitempty"`
}

// ParseHTTPRequest parses a raw HTTP request string. Header keys are
// normalized to lower case for case-insensitive lookups; Content-Length is
// recomputed so users can edit the body without counting bytes.
func ParseHTTPRequest(raw string) (*ParsedInput, error) {
	raw = strings.TrimLeft(raw, " \t\r\n")
	normalized := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\n", "\r\n")

	if headerEnd := strings.Index(normalized, "\r\n\r\n"); headerEnd >= 0 {
		head := normalized[:headerEnd]
		body := normalized[headerEnd+4:]
		lines := strings.Split(head, "\r\n")
		hasCL := false
		for i, l := range lines {
			if strings.HasPrefix(strings.ToLower(l), "content-length:") {
				lines[i] = fmt.Sprintf("Content-Length: %d", len(body))
				hasCL = true
			}
		}
		if !hasCL && len(body) > 0 {
			lines = append(lines, fmt.Sprintf("Content-Length: %d", len(body)))
		}
		normalized = strings.Join(lines, "\r\n") + "\r\n\r\n" + body
	}

	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(normalized)))
	if err != nil {
		return nil, fmt.Errorf("parse HTTP request: %w", err)
	}
	defer req.Body.Close()

	headers := make(map[string][]string, len(req.Header))
	for k, v := range req.Header {
		headers[strings.ToLower(k)] = v
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	rawBody := string(bodyBytes)

	var body interface{}
	bodyParsed := false
	if len(strings.TrimSpace(rawBody)) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err == nil {
			bodyParsed = true
		}
	}

	return &ParsedInput{
		Method:     req.Method,
		URL:        req.URL.String(),
		Header:     headers,
		Body:       body,
		RawBody:    rawBody,
		BodyParsed: bodyParsed,
	}, nil
}

func headerMatch(args ...ref.Val) ref.Val {
	if len(args) != 3 {
		return types.NewErr("match() expects 3 arguments")
	}
	hdr, ok := args[0].(traits.Mapper)
	if !ok {
		return types.NewErr("match() must be called on a header map")
	}
	key, ok := args[1].Value().(string)
	if !ok {
		return types.NewErr("match() key must be a string")
	}
	want, ok := args[2].Value().(string)
	if !ok {
		return types.NewErr("match() value must be a string")
	}
	v, found := hdr.Find(types.String(strings.ToLower(key)))
	if !found {
		return types.Bool(false)
	}
	lst, ok := v.(traits.Lister)
	if !ok {
		return types.Bool(false)
	}
	it := lst.Iterator()
	for it.HasNext() == types.Bool(true) {
		got, ok := it.Next().Value().(string)
		if ok && got == want {
			return types.Bool(true)
		}
	}
	return types.Bool(false)
}

func headerCanonical(args ...ref.Val) ref.Val {
	if len(args) != 2 {
		return types.NewErr("canonical() expects 2 arguments")
	}
	hdr, ok := args[0].(traits.Mapper)
	if !ok {
		return types.NewErr("canonical() must be called on a header map")
	}
	key, ok := args[1].Value().(string)
	if !ok {
		return types.NewErr("canonical() key must be a string")
	}
	v, found := hdr.Find(types.String(strings.ToLower(key)))
	if !found {
		return types.String("")
	}
	lst, ok := v.(traits.Lister)
	if !ok {
		return types.String("")
	}
	it := lst.Iterator()
	if it.HasNext() == types.Bool(true) {
		if s, ok := it.Next().Value().(string); ok {
			return types.String(s)
		}
	}
	return types.String("")
}

// Evaluate compiles and evaluates expression against the parsed HTTP request.
func Evaluate(expression string, parsed *ParsedInput) (interface{}, error) {
	headerType := cel.MapType(cel.StringType, cel.ListType(cel.StringType))

	env, err := cel.NewEnv(
		cel.Variable("body", cel.DynType),
		cel.Variable("header", headerType),
		cel.Variable("requestURL", cel.StringType),
		cel.Variable("method", cel.StringType),
		cel.Function("match",
			cel.MemberOverload("header_match_string_string",
				[]*cel.Type{headerType, cel.StringType, cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(headerMatch),
			),
		),
		cel.Function("canonical",
			cel.MemberOverload("header_canonical_string",
				[]*cel.Type{headerType, cel.StringType},
				cel.StringType,
				cel.FunctionBinding(headerCanonical),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile: %s", issues.Err().Error())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program: %w", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"body":       parsed.Body,
		"header":     parsed.Header,
		"requestURL": parsed.URL,
		"method":     parsed.Method,
	})
	if err != nil {
		return nil, fmt.Errorf("eval: %w", err)
	}
	return out.Value(), nil
}

// Handle is the shared HTTP handler used by both the local dev server and the
// Vercel serverless function. It writes a JSON Response on every code path.
func Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Error: "invalid JSON: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.CelExpression) == "" {
		writeJSON(w, http.StatusOK, Response{Error: "celExpression is required"})
		return
	}
	if strings.TrimSpace(req.HttpRequest) == "" {
		writeJSON(w, http.StatusOK, Response{Error: "httpRequest is required"})
		return
	}

	parsed, err := ParseHTTPRequest(req.HttpRequest)
	if err != nil {
		writeJSON(w, http.StatusOK, Response{Error: err.Error()})
		return
	}

	resp := Response{ParsedInput: parsed}
	if result, err := Evaluate(req.CelExpression, parsed); err != nil {
		resp.Error = err.Error()
	} else {
		resp.Result = result
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
