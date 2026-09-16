// Package http exposes customer-owned saved payment method metadata endpoints.
package http

import (
	"errors"
	"net/http"

	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

type createRequest struct {
	Brand       domain.CardBrand `json:"brand"`
	Last4       string           `json:"last4"`
	ExpiryMonth int              `json:"expiryMonth"`
	ExpiryYear  int              `json:"expiryYear"`
	HolderName  string           `json:"holderName"`
}

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	methods, err := handler.service.List(r.Context(), actor)
	if writeError(w, r, err) {
		return
	}
	items := make([]paymentMethodResponse, len(methods))
	for index, method := range methods {
		items[index] = response(method)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	var request createRequest
	if httpx.DecodeJSON(w, r, &request) != nil {
		httpx.WriteRequestError(w, r, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	method, err := handler.service.Create(r.Context(), actor, domain.CreateInput{
		Brand: request.Brand, Last4: request.Last4, ExpiryMonth: request.ExpiryMonth,
		ExpiryYear: request.ExpiryYear, HolderName: request.HolderName,
	})
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response(method))
}

func (handler *Handler) SetDefault(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "paymentMethodID", "payment_method_id")
	if !ok {
		return
	}
	method, err := handler.service.SetDefault(r.Context(), actor, id)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response(method))
}

func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "paymentMethodID", "payment_method_id")
	if !ok {
		return
	}
	if writeError(w, r, handler.service.Delete(r.Context(), actor, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func actorFromRequest(w http.ResponseWriter, r *http.Request) (application.Actor, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return application.Actor{}, false
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	return application.Actor{ID: user.ID, Role: role}, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "only customers can manage payment methods")
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "payment method was not found")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "payment method metadata is invalid")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "payment_methods_unavailable", "could not complete the payment method request")
	}
	return true
}
