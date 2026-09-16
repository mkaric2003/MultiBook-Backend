package http

import (
	"errors"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
	"net/http"
	"strings"
)

type Handler struct{ service *application.Service }

func NewHandler(s *application.Service) *Handler { return &Handler{s} }
func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	u, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	var input domain.ReplaceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	role := ""
	if u.Role != nil {
		role = string(*u.Role)
	}
	items, err := h.service.Replace(r.Context(), u.ID, role, id, input)
	if handle(w, r, err) {
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": items})
}
func handle(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, application.ErrNotFound) {
		httpx.WriteRequestError(w, r, 404, "not_found", "business was not found")
	} else if errors.Is(err, application.ErrForbidden) {
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	} else if errors.Is(err, application.ErrValidation) {
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	} else {
		httpx.WriteRequestError(w, r, 500, "business_media_unavailable", "could not complete the media request")
	}
	return true
}
