package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestPathUUID(t *testing.T) {
	want := uuid.New()
	recorder := httptest.NewRecorder()
	got, ok := PathUUID(recorder, requestWithPathParam("businessID", want.String()), "businessID", "business_id")
	if !ok || got != want {
		t.Fatalf("PathUUID() = %s, %v; want %s, true", got, ok, want)
	}

	recorder = httptest.NewRecorder()
	got, ok = PathUUID(recorder, requestWithPathParam("businessID", "invalid"), "businessID", "business_id")
	if ok || got != uuid.Nil || recorder.Code != 422 {
		t.Fatalf("invalid PathUUID() = %s, %v, status %d", got, ok, recorder.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"]["code"] != "validation_error" || body["error"]["message"] != "business_id must be a UUID" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func requestWithPathParam(name, value string) *http.Request {
	request := httptest.NewRequest("GET", "/", nil)
	route := chi.NewRouteContext()
	route.URLParams.Add(name, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}
