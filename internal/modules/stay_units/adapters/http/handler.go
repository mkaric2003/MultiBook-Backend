// Package http is the HTTP adapter for the stay_units module.
package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_units/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, _, ok := requestContext(w, r, false)
	if !ok {
		return
	}
	var input domain.CreateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return
	}
	result, err := h.service.Create(r.Context(), actorID, role, businessID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, _, ok := requestContext(w, r, false)
	if !ok {
		return
	}
	result, err := h.service.List(r.Context(), actorID, role, businessID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": result})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, unitTypeID, ok := requestContext(w, r, true)
	if !ok {
		return
	}
	var input domain.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return
	}
	result, err := h.service.Update(r.Context(), actorID, role, businessID, unitTypeID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, unitTypeID, ok := requestContext(w, r, true)
	if !ok {
		return
	}
	if writeError(w, r, h.service.Archive(r.Context(), actorID, role, businessID, unitTypeID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func requestContext(w http.ResponseWriter, r *http.Request, requireUnitType bool) (string, string, uuid.UUID, uuid.UUID, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, false
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, false
	}
	unitTypeID := uuid.Nil
	if requireUnitType {
		unitTypeID, ok = httpx.PathUUID(w, r, "unitTypeID", "unit_type_id")
		if !ok {
			return "", "", uuid.Nil, uuid.Nil, false
		}
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	return user.ID, role, businessID, unitTypeID, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "stay or unit type was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "stay_unit_unavailable", "could not complete the stay unit request")
	}
	return true
}
