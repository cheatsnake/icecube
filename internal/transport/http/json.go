package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Message struct {
	Message string `json:"message"`
}

func jsonBodyParse[T any](r *http.Request) (*T, error) {
	rawBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if len(rawBytes) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}

	var jsonData T
	if err := json.Unmarshal(rawBytes, &jsonData); err != nil {
		return nil, err
	}

	return &jsonData, nil
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "JSON encoding failed", http.StatusInternalServerError)
	}
}

func jsonMessage(w http.ResponseWriter, code int, body string) {
	msg := Message{Message: body}

	w.WriteHeader(code)
	jsonResponse(w, msg)
}

func jsonOK(w http.ResponseWriter, body string) {
	jsonMessage(w, http.StatusOK, body)
}

func jsonBadRequest(w http.ResponseWriter, body string) {
	jsonMessage(w, http.StatusBadRequest, body)
}

func jsonNotFound(w http.ResponseWriter, body string) {
	jsonMessage(w, http.StatusNotFound, body)
}

func jsonMethodNotAllowed(w http.ResponseWriter) {
	jsonMessage(w, http.StatusMethodNotAllowed, "Method not allowed")
}

func jsonInternalError(w http.ResponseWriter, body string) {
	jsonMessage(w, http.StatusInternalServerError, body)
}
