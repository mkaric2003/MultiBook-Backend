package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type httpStub struct{ err error }

func (s *httpStub) Create(_ context.Context, customerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Review, error) {
	return domain.Review{ID: uuid.New(), CustomerID: customerID, BusinessID: businessID, CustomerName: "Customer", Rating: input.Rating, Comment: input.Comment}, s.err
}
func (s *httpStub) HasReview(context.Context, string, uuid.UUID) (bool, error) { return true, s.err }
func (s *httpStub) List(context.Context, uuid.UUID, int, int) ([]domain.Review, error) {
	return []domain.Review{{ID: uuid.New(), CustomerName: "Customer", Rating: 5}}, s.err
}

func TestReviewsHTTPContract(t *testing.T) {
	stub := &httpStub{}
	handler := NewHandler(application.NewService(stub, stub))
	businessID := uuid.NewString()

	recorder := httptest.NewRecorder()
	handler.Create(recorder, request(http.MethodPost, businessID, `{"sourceId":"`+uuid.NewString()+`","type":"stay","rating":5,"comment":"Great"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.HasReview(recorder, request(http.MethodGet, businessID, ""))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"hasReview":true`)) {
		t.Fatalf("has status %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.List(recorder, request(http.MethodGet, businessID, ""))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"customerName":"Customer"`)) {
		t.Fatalf("list status %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestReviewsHTTPErrors(t *testing.T) {
	handler := NewHandler(application.NewService(&httpStub{}, &httpStub{}))
	recorder := httptest.NewRecorder()
	handler.Create(recorder, request(http.MethodPost, uuid.NewString(), `{"sourceId":"invalid","type":"stay","rating":5}`))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid source status %d", recorder.Code)
	}
}

func request(method, businessID, body string) *http.Request {
	role := usersdomain.UserRoleCustomer
	r := usershttp.WithCurrentUser(httptest.NewRequest(method, "/businesses/"+businessID+"/reviews", bytes.NewBufferString(body)), usersdomain.User{ID: "customer", Role: &role})
	r.Header.Set("Content-Type", "application/json")
	route := chi.NewRouteContext()
	route.URLParams.Add("businessID", businessID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
}
