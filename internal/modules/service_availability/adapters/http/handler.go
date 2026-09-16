// Package http is the HTTP adapter for staff availability.
package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_availability/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) CreateWeekly(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, _, ok := ids(w, r, "")
	if !ok {
		return
	}
	var input domain.CreateWeeklyInput
	if httpx.DecodeJSON(w, r, &input) != nil {
		invalidBody(w, r)
		return
	}
	item, err := h.service.CreateWeekly(r.Context(), actorID, role, businessID, staffID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, item)
}
func (h *Handler) ListWeekly(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, _, ok := ids(w, r, "")
	if !ok {
		return
	}
	items, err := h.service.ListWeekly(r.Context(), actorID, role, businessID, staffID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *Handler) UpdateWeekly(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, id, ok := ids(w, r, "weeklyAvailabilityID")
	if !ok {
		return
	}
	var input domain.UpdateWeeklyInput
	if httpx.DecodeJSON(w, r, &input) != nil {
		invalidBody(w, r)
		return
	}
	item, err := h.service.UpdateWeekly(r.Context(), actorID, role, businessID, staffID, id, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}
func (h *Handler) DeleteWeekly(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, id, ok := ids(w, r, "weeklyAvailabilityID")
	if !ok {
		return
	}
	if writeError(w, r, h.service.DeleteWeekly(r.Context(), actorID, role, businessID, staffID, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, _, ok := ids(w, r, "")
	if !ok {
		return
	}
	var input domain.CreateBlockInput
	if httpx.DecodeJSON(w, r, &input) != nil {
		invalidBody(w, r)
		return
	}
	item, err := h.service.CreateBlock(r.Context(), actorID, role, businessID, staffID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, item)
}
func (h *Handler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, _, ok := ids(w, r, "")
	if !ok {
		return
	}
	items, err := h.service.ListBlocks(r.Context(), actorID, role, businessID, staffID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *Handler) UpdateBlock(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, id, ok := ids(w, r, "availabilityBlockID")
	if !ok {
		return
	}
	var input domain.UpdateBlockInput
	if httpx.DecodeJSON(w, r, &input) != nil {
		invalidBody(w, r)
		return
	}
	item, err := h.service.UpdateBlock(r.Context(), actorID, role, businessID, staffID, id, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}
func (h *Handler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	actorID, role, businessID, staffID, id, ok := ids(w, r, "availabilityBlockID")
	if !ok {
		return
	}
	if writeError(w, r, h.service.DeleteBlock(r.Context(), actorID, role, businessID, staffID, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ids(w http.ResponseWriter, r *http.Request, resourceParam string) (string, string, uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	staffID, ok := httpx.PathUUID(w, r, "staffID", "staff_id")
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	id := uuid.Nil
	if resourceParam != "" {
		id, ok = httpx.PathUUID(w, r, resourceParam, strings.TrimSuffix(strings.TrimSuffix(resourceParam, "ID"), "ID"))
		if !ok {
			return "", "", uuid.Nil, uuid.Nil, uuid.Nil, false
		}
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	return user.ID, role, businessID, staffID, id, true
}
func invalidBody(w http.ResponseWriter, r *http.Request) {
	httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
}
func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, 404, "not_found", "service staff or availability resource was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	default:
		httpx.WriteRequestError(w, r, 500, "service_availability_unavailable", "could not complete service availability request")
	}
	return true
}
