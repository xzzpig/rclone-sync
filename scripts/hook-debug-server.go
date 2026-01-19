package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

func main() {
	port := "8078"
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().Format("15:04:05")

		dump, _ := httputil.DumpRequest(r, false)

		fmt.Printf("\n\033[1;34m[%s] >>>>> New Hook Received from %s <<<<<\033[0m\n", now, r.RemoteAddr)
		fmt.Printf("\033[33m%s\033[0m", string(dump))

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Error reading body: %v\n", err)
		} else if len(body) > 0 {
			fmt.Println("Body:")
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				var prettyJSON bytes.Buffer
				if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
					fmt.Println(prettyJSON.String())
				} else {
					fmt.Println(string(body))
				}
			} else {
				fmt.Println(string(body))
			}
		} else {
			fmt.Println("Body: <empty>")
		}

		fmt.Printf("\033[1;34m[%s] ^^^^^ End of Hook ^^^^^\033[0m\n", now)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Hook received and logged\n")
	})

	log.Printf("Hook debug server is listening on http://localhost:%s", port)
	log.Printf("Point your hooks to this URL for testing.")
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
