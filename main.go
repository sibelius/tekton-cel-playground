// Local development server. In production, the same evaluator is exposed as a
// Vercel serverless function — see api/evaluate.go.
package main

import (
	"fmt"
	"log"
	"net/http"

	"cel-playground/internal/celeval"
)

func main() {
	http.HandleFunc("/api/evaluate", celeval.Handle)
	addr := ":3002"
	fmt.Printf("CEL playground server listening on http://localhost%s/api/evaluate\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
