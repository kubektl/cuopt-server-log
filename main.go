package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type SaveRequest struct {
	Timestamp            string      `json:"timestamp"`
	M                    int         `json:"m"`
	FullSolutionResponse interface{} `json:"full_solution_response"`
	Status               string      `json:"status"`
	BestKnownMinScore    *int        `json:"best_known_min_score,omitempty"`
	BestBound            *float64    `json:"best_bound,omitempty"`
	VarsFound            interface{} `json:"vars_found,omitempty"`
}

type Response struct {
	Status bool `json:"status"`
}

func main() {
	http.HandleFunc("/save", saveHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Go server running on :%s\n", port)
	fmt.Printf("POST your result to http://localhost:%s/save\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Limit request body size to avoid DoS via huge payloads (10 MB)
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req SaveRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Ensure results directory exists
	if err := os.MkdirAll("results", 0o755); err != nil {
		log.Printf("Failed to create results directory: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Filename with timestamp and m
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join("results", fmt.Sprintf("xubit_m%d_%s.json", req.M, timestamp))

	if err := os.WriteFile(filename, body, 0o644); err != nil {
		log.Printf("Failed to save: %v", err)
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}

	log.Printf("Saved: %s (m=%d)", filename, req.M)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Response{Status: true})
}