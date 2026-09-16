// Package http exposes Saved businesses HTTP endpoints.
package http

import (
	"errors"
	"net/http"

	discoveryresponse "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/http/response"
	"github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.Save(r.Context(), actor, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.Remove(r.Context(), actor, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) IsSaved(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	saved, err := h.service.IsSaved(r.Context(), actor, id)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"isSaved": saved})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), actor)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": discoveryresponse.Summaries(items)})
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

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrBusinessNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "business was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "business_id is required")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "saved_businesses_unavailable", "could not complete the Saved businesses request")
	}
	return true
}
