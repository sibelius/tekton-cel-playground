# tekton-cel-playground

A small playground that mirrors the behavior of [Tekton Triggers'
`cel-eval`](https://github.com/tektoncd/triggers/blob/main/cmd/cel-eval/cmd/root.go):
paste a raw HTTP request and a [CEL](https://github.com/google/cel-spec)
expression, hit Evaluate, and see the result.

The CEL environment exposes the same bindings Tekton Triggers does
(see [docs](https://tekton.dev/docs/triggers/cel_expressions/)):

| Binding         | Type                                  | Notes                                       |
| --------------- | ------------------------------------- | ------------------------------------------- |
| `body`          | dyn (parsed JSON body)                | `null` when the body is not valid JSON      |
| `header`        | `map<string, list<string>>`           | Keys are lower-cased for case-insensitivity |
| `requestURL`    | string                                | Path + query from the request line          |
| `method`        | string                                |                                             |

Helper member functions:

- `header.match(key, value)` — true if any value of `key` (case-insensitive) equals `value`.
- `header.canonical(key)` — first value of `key` (case-insensitive), or `""`.

## Run it

```sh
pnpm install
pnpm dev
```

`pnpm dev` runs both the Go API (`:3002`, hot-reloaded by nodemon) and the Next
frontend (`:3000`, proxies `/api/evaluate` to the Go server) in parallel.
Open <http://localhost:3000> and pick one of the example presets.

Other scripts:

| Script              | What it does                                 |
| ------------------- | -------------------------------------------- |
| `pnpm dev`          | Both servers, both hot-reloaded              |
| `pnpm dev:web`      | Only Next.js dev server                      |
| `pnpm dev:api`      | Only Go API, watching `*.go` and `go.mod`    |
| `pnpm dev:api:once` | Run the Go API once with `go run main.go`    |
| `pnpm build`        | Production build of the frontend             |

## Try it from curl

```sh
curl -sS -X POST http://localhost:3002/api/evaluate \
  -H 'Content-Type: application/json' \
  -d '{
    "celExpression": "header.match(\"x-github-event\", \"push\") && body.repository.name == \"hello\"",
    "httpRequest": "POST /webhook HTTP/1.1\nContent-Type: application/json\nX-GitHub-Event: push\n\n{\"repository\":{\"name\":\"hello\"}}"
  }'
```

`Content-Length` is recomputed server-side, so you can edit the body without
counting bytes.

## Deploy to Vercel

The repo is set up for Vercel out of the box:

- `src/app/` is the Next.js frontend (auto-detected).
- `api/evaluate.go` is a Vercel Go serverless function exposed at
  `/api/evaluate` — it shares its evaluator with the local server via
  `internal/celeval`.
- `next.config.ts`'s rewrite to `localhost:3002` is dev-only (gated on
  `NODE_ENV`), so in production both routes are served by Vercel.

```sh
# one-off
pnpm dlx vercel        # link the project, then 'vercel deploy'
pnpm dlx vercel --prod # production deploy
```

Or push to a Git remote and connect the repo in the Vercel dashboard — no
extra configuration needed. The Go runtime is detected automatically from
`api/*.go`; Vercel will run `go mod download` and build the function.

> First request after idle has a ~300–500 ms cold start (cel-go init). For
> always-warm latency, deploy the Go server separately on Fly.io / Railway and
> point the dev rewrite (or production rewrite) at its public URL.

## Project layout

```
api/evaluate.go         Vercel serverless function (production)
main.go                 Local dev HTTP server
internal/celeval/       Shared parser + CEL evaluator
src/app/                Next.js frontend
```

## Contributing

Check [CONTRIBUTING.md](./CONTRIBUTING.md) for the steps.
