package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

func TestRequireCurrentUser(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	if _, ok := RequireCurrentUser(recorder, request); ok || recorder.Code != http.StatusInternalServerError {
		t.Fatalf("missing user: ok=%v status=%d", ok, recorder.Code)
	}

	want := domain.User{ID: "customer"}
	request = WithCurrentUser(httptest.NewRequest(http.MethodGet, "/", nil), want)
	recorder = httptest.NewRecorder()
	got, ok := RequireCurrentUser(recorder, request)
	if !ok || got.ID != want.ID || recorder.Code != http.StatusOK {
		t.Fatalf("provisioned user: got=%#v ok=%v status=%d", got, ok, recorder.Code)
	}
}
