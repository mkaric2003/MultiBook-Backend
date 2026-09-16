package http

import (
	"errors"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/services/domain"
	users "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
	"net/http"
	"strings"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service} }
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	a, role, id, ok := ctx(w, r)
	if !ok {
		return
	}
	var in domain.UpsertInput
	if httpx.DecodeJSON(w, r, &in) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	v, e := h.service.Upsert(r.Context(), a, role, id, in)
	if write(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 200, v)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	a, role, id, ok := ctx(w, r)
	if !ok {
		return
	}
	v, e := h.service.Get(r.Context(), a, role, id)
	if write(w, r, e) {
		return
	}
	httpx.WriteJSON(w, 200, v)
}
func ctx(w http.ResponseWriter, r *http.Request) (string, string, uuid.UUID, bool) {
	u, ok := users.RequireCurrentUser(w, r)
	if !ok {
		return "", "", uuid.Nil, false
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return "", "", uuid.Nil, false
	}
	role := ""
	if u.Role != nil {
		role = string(*u.Role)
	}
	return u.ID, role, id, true
}
func write(w http.ResponseWriter, r *http.Request, e error) bool {
	if e == nil {
		return false
	}
	if errors.Is(e, application.ErrNotFound) {
		httpx.WriteRequestError(w, r, 404, "not_found", "service business or details were not found")
	} else if errors.Is(e, application.ErrForbidden) {
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	} else if errors.Is(e, application.ErrValidation) {
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(e.Error(), "validation failed: "))
	} else {
		httpx.WriteRequestError(w, r, 500, "service_unavailable", "could not complete service request")
	}
	return true
}
