package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// runHealthcheck probes the local /healthz endpoint and exits with code 0 on
// success or 1 on failure. This is used as the container healthcheck command
// so that distroless images (which have no shell, wget, or curl) can report
// health to the orchestrator.
func runHealthcheck() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/healthz", port))
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		os.Exit(0)
	}
	os.Exit(1)
}
