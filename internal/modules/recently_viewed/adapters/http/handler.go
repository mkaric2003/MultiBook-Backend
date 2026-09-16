// Package http exposes Recently Viewed HTTP endpoints.
package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	discoveryresponse "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/http/response"
	"github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.Record(r.Context(), actor, businessID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	businessType, ok := businessTypeFromRequest(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), actor, businessType, limit)
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

func businessTypeFromRequest(w http.ResponseWriter, r *http.Request) (domain.BusinessType, bool) {
	switch r.URL.Query().Get("type") {
	case "stays":
		return domain.BusinessTypeStay, true
	case "services":
		return domain.BusinessTypeService, true
	default:
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "type must be stays or services")
		return "", false
	}
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
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "recently viewed input is invalid")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "recently_viewed_unavailable", "could not complete the Recently Viewed request")
	}
	return true
}
