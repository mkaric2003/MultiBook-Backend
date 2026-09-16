package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/domain"
	users "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service} }
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	a, role, b, _, ok := contextIDs(w, r, false)
	if !ok {
		return
	}
	var in domain.CreateInput
	if httpx.DecodeJSON(w, r, &in) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	v, e := h.service.Create(r.Context(), a, role, b, in)
	if write(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 201, v)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	a, role, b, _, ok := contextIDs(w, r, false)
	if !ok {
		return
	}
	v, e := h.service.List(r.Context(), a, role, b)
	if write(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": v})
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	a, role, b, id, ok := contextIDs(w, r, true)
	if !ok {
		return
	}
	var in domain.UpdateInput
	if httpx.DecodeJSON(w, r, &in) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	v, e := h.service.Update(r.Context(), a, role, b, id, in)
	if write(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 200, v)
}
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	a, role, b, id, ok := contextIDs(w, r, true)
	if !ok {
		return
	}
	if write(w, r, h.service.Archive(r.Context(), a, role, b, id)) {
		return
	}
	w.WriteHeader(204)
}
func contextIDs(w http.ResponseWriter, r *http.Request, needOffering bool) (string, string, uuid.UUID, uuid.UUID, bool) {
	u, ok := users.RequireCurrentUser(w, r)
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, false
	}
	b, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return "", "", uuid.Nil, uuid.Nil, false
	}
	id := uuid.Nil
	if needOffering {
		id, ok = httpx.PathUUID(w, r, "offeringID", "offering_id")
		if !ok {
			return "", "", uuid.Nil, uuid.Nil, false
		}
	}
	role := ""
	if u.Role != nil {
		role = string(*u.Role)
	}
	return u.ID, role, b, id, true
}
func write(w http.ResponseWriter, r *http.Request, e error) bool {
	if e == nil {
		return false
	}
	if errors.Is(e, application.ErrNotFound) {
		httpx.WriteRequestError(w, r, 404, "not_found", "service business or offering was not found")
	} else if errors.Is(e, application.ErrForbidden) {
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	} else if errors.Is(e, application.ErrValidation) {
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(e.Error(), "validation failed: "))
	} else {
		httpx.WriteRequestError(w, r, 500, "service_offering_unavailable", "could not complete service offering request")
	}
	return true
}
