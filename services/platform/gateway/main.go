package main

import (
	"fmt"
	"net/http"
	"os"
)

// M0 stub — replaced by the real implementation in its milestone (ROADMAP.md).
func main() {
	const port = "8080"
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"status\":\"ok\"}")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s (stub)\n", os.Getenv("OTEL_SERVICE_NAME"))
	})
	fmt.Println("stub listening on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
