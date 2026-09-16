package http

import (
	"errors"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_staff/domain"
	users "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
	"net/http"
	"strings"
)

type Handler struct{ service *application.Service }

func NewHandler(s *application.Service) *Handler { return &Handler{s} }
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	a, role, b, ok := actor(w, r)
	if !ok {
		return
	}
	var i domain.CreateInput
	if httpx.DecodeJSON(w, r, &i) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	v, e := h.service.Create(r.Context(), a, role, b, i)
	if err(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 201, v)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, role, b, ok := actor(w, r)
	if !ok {
		return
	}
	v, e := h.service.List(r.Context(), a, role, b)
	if err(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": v})
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	a, role, b, id, ok := actorStaff(w, r)
	if !ok {
		return
	}
	var i domain.UpdateInput
	if httpx.DecodeJSON(w, r, &i) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	v, e := h.service.Update(r.Context(), a, role, b, id, i)
	if err(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 200, v)
}
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	a, role, b, id, ok := actorStaff(w, r)
	if !ok {
		return
	}
	if err(w, r, h.service.Archive(r.Context(), a, role, b, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func actor(w http.ResponseWriter, r *http.Request) (string, string, uuid.UUID, bool) {
	u, ok := users.RequireCurrentUser(w, r)
	if !ok {
		return "", "", uuid.Nil, false
	}
	b, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return "", "", uuid.Nil, false
	}
	role := ""
	if u.Role != nil {
		role = string(*u.Role)
	}
	return u.ID, role, b, true
}
func actorStaff(w http.ResponseWriter, r *http.Request) (string, string, uuid.UUID, uuid.UUID, bool) {
	a, role, b, ok := actor(w, r)
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, false
	}
	id, ok := httpx.PathUUID(w, r, "staffID", "staff_id")
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, false
	}
	return a, role, b, id, true
}
func err(w http.ResponseWriter, r *http.Request, e error) bool {
	if e == nil {
		return false
	}
	switch {
	case errors.Is(e, application.ErrNotFound):
		httpx.WriteRequestError(w, r, 404, "not_found", "service business or staff member was not found")
	case errors.Is(e, application.ErrForbidden):
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	case errors.Is(e, application.ErrValidation):
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(e.Error(), "validation failed: "))
	default:
		httpx.WriteRequestError(w, r, 500, "service_staff_unavailable", "could not complete service staff request")
	}
	return true
}
