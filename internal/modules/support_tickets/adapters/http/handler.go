// Package http exposes authenticated support ticket endpoints.
package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

type createRequest struct {
	Category domain.Category `json:"category"`
	Subject  string          `json:"subject"`
	Message  string          `json:"message"`
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	var request createRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteRequestError(w, r, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	ticket, err := handler.service.Create(r.Context(), actor, domain.CreateInput{
		Category: request.Category, Subject: request.Subject, Message: request.Message,
	})
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response(ticket))
}

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	offset, pageSize, ok := pageInput(w, r)
	if !ok {
		return
	}
	page, err := handler.service.List(r.Context(), actor, offset, pageSize)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pageResponseFrom(page))
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

func pageInput(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	offset := 0
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "cursor must be a non-negative integer")
			return 0, 0, false
		}
		offset = parsed
	}
	pageSize := 0
	if value := r.URL.Query().Get("page_size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "page_size must be an integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	return offset, pageSize, true
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "only customers can access support tickets")
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "customer profile was not found")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "support ticket input is invalid")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "support_tickets_unavailable", "could not complete the support ticket request")
	}
	return true
}
