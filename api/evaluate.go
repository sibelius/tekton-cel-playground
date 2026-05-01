// Vercel serverless function entry point. Vercel's Go runtime auto-detects
// any `Handler` exported from a file under /api and exposes it at the matching
// path — so this file is served at /api/evaluate.
//
// Docs: https://vercel.com/docs/functions/runtimes/go
package handler

import (
	"net/http"

	"cel-playground/internal/celeval"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	celeval.Handle(w, r)
}
