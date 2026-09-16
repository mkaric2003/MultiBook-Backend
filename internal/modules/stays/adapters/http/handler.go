// Package http is the HTTP adapter for the stays module.
package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stays/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	var input domain.UpsertInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return
	}
	result, err := h.service.Upsert(r.Context(), actorID, role, businessID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) GetOwned(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	result, err := h.service.GetOwned(r.Context(), actorID, role, businessID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func requestContext(w http.ResponseWriter, r *http.Request) (string, string, uuid.UUID, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return "", "", uuid.Nil, false
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return "", "", uuid.Nil, false
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	return user.ID, role, businessID, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "stay business or details were not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "stay_unavailable", "could not complete the stay request")
	}
	return true
}
