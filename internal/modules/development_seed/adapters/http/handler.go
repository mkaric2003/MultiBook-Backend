package http

import (
	"context"
	"errors"
	"github.com/mkaric2003/multibook-backend/internal/modules/development_seed/application"
	users "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
	"net/http"
)

type Handler struct{ service *application.Service }

func NewHandler(s *application.Service) *Handler                   { return &Handler{s} }
func (h *Handler) Stays(w http.ResponseWriter, r *http.Request)    { h.seed(w, r, h.service.Stays) }
func (h *Handler) Services(w http.ResponseWriter, r *http.Request) { h.seed(w, r, h.service.Services) }
func (h *Handler) seed(w http.ResponseWriter, r *http.Request, operation func(rctx context.Context, actor, role string) (int, error)) {
	user, ok := users.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	count, err := operation(r.Context(), user.ID, role)
	if err != nil {
		if errors.Is(err, application.ErrForbidden) {
			httpx.WriteRequestError(w, r, 403, "forbidden", "development seed endpoints require APP_ENV=development and a provider role")
		} else {
			httpx.WriteRequestError(w, r, 500, "development_seed_unavailable", "could not seed development data")
		}
		return
	}
	httpx.WriteJSON(w, 201, map[string]int{"created": count})
}
