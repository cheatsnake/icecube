package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/cheatsnake/icecube/internal/domain/errs"
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

func jsonInternalError(w http.ResponseWriter, body string) {
	jsonMessage(w, http.StatusInternalServerError, body)
}

// handleError checks the error type and returns appropriate HTTP status
func handleError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	// Get the meaningful message from the error chain
	msg := extractErrorMessage(err)

	if errors.Is(err, errs.ErrNotFound) {
		jsonNotFound(w, msg)
		return true
	}

	if errors.Is(err, errs.ErrAlreadyExists) {
		jsonMessage(w, http.StatusConflict, msg)
		return true
	}

	if errors.Is(err, errs.ErrInvalidInput) {
		jsonBadRequest(w, msg)
		return true
	}

	// Default to internal error for unknown errors
	jsonInternalError(w, msg)
	return true
}

// extractErrors extracts all errors from a joined error chain
func extractErrors(err error) []error {
	var joinErr interface {
		Unwrap() []error
	}

	if errors.As(err, &joinErr) {
		return joinErr.Unwrap()
	}
	return []error{err}
}

// extractErrorMessage extracts the most meaningful message from an error (including joined errors)
func extractErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errs := extractErrors(err)
	// Return the last error message (most specific one in the chain)
	if len(errs) > 0 {
		return errs[len(errs)-1].Error()
	}

	return err.Error()
}
