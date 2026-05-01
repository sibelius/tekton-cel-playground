// Local development server. The same Handler is reused by the Vercel
// serverless function in api/evaluate.go.
package main

import (
	"fmt"
	"log"
	"net/http"

	"cel-playground/api"
)

func main() {
	http.HandleFunc("/api/evaluate", api.Handler)
	addr := ":3002"
	fmt.Printf("CEL playground server listening on http://localhost%s/api/evaluate\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
