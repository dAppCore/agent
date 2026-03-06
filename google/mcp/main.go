package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type GoTestRequest struct {
	Filter   string `json:"filter,omitempty"`
	Coverage bool   `json:"coverage,omitempty"`
}

type GoTestResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type DevHealthResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type DevCommitRequest struct {
	Message string   `json:"message"`
	Repos   []string `json:"repos,omitempty"`
}

type DevCommitResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func goTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var req GoTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	args := []string{"go", "test"}
	if req.Filter != "" {
		args = append(args, "-run", req.Filter)
	}
	if req.Coverage {
		args = append(args, "-cover")
	}

	cmd := exec.Command("core", args...)
	output, err := cmd.CombinedOutput()

	resp := GoTestResponse{
		Output: string(output),
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func devHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	cmd := exec.Command("core", "dev", "health")
	output, err := cmd.CombinedOutput()

	resp := DevHealthResponse{
		Output: string(output),
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func devCommitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var req DevCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	args := []string{"dev", "commit", "-m", req.Message}
	if len(req.Repos) > 0 {
		args = append(args, "--repos", strings.Join(req.Repos, ","))
	}

	cmd := exec.Command("core", args...)
	output, err := cmd.CombinedOutput()

	resp := DevCommitResponse{
		Output: string(output),
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/core_go_test", goTestHandler)
	http.HandleFunc("/core_dev_health", devHealthHandler)
	http.HandleFunc("/core_dev_commit", devCommitHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "MCP Server is running")
	})

	log.Println("Starting MCP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("could not start server: %s\n", err)
	}
}
