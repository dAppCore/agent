package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	// Get the absolute path to the testdata directory
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testdataPath := filepath.Join(wd, "testdata")

	// Add the absolute path to the PATH
	os.Setenv("PATH", testdataPath+":"+os.Getenv("PATH"))
	m.Run()
}
func TestGoTestHandler(t *testing.T) {
	reqBody := GoTestRequest{
		Filter:   "TestMyFunction",
		Coverage: true,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "/core_go_test", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(goTestHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var resp GoTestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("handler returned an unexpected error: %v", resp.Error)
	}
}

func TestDevHealthHandler(t *testing.T) {
	req, err := http.NewRequest("POST", "/core_dev_health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(devHealthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var resp DevHealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("handler returned an unexpected error: %v", resp.Error)
	}
}

func TestDevCommitHandler(t *testing.T) {
	reqBody := DevCommitRequest{
		Message: "test commit",
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "/core_dev_commit", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(devCommitHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var resp DevCommitResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("handler returned an unexpected error: %v", resp.Error)
	}
}
