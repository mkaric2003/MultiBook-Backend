package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// DecodeJSON reads exactly one JSON value, rejects unknown fields, and limits
// request bodies. It is a transport primitive shared by HTTP adapters.
func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("decode JSON request: body must contain one JSON value")
	}
	return nil
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	writeError(w, status, code, message, "")
}

func WriteRequestError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeError(w, status, code, message, middleware.GetReqID(r.Context()))
}

func writeError(w http.ResponseWriter, status int, code, message, requestID string) {
	errorBody := map[string]string{
		"code":    code,
		"message": message,
	}
	if requestID != "" {
		errorBody["request_id"] = requestID
	}
	WriteJSON(w, status, map[string]any{
		"error": errorBody,
	})
}
