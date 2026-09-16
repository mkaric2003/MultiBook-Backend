// Package http is the HTTP adapter for the users module.
package http

import (
	"errors"
	"net/http"

	"github.com/mkaric2003/multibook-backend/internal/modules/users/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/auth"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := RequireCurrentUser(w, r)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "authenticated_user_unavailable", "authenticated user is unavailable")
		return
	}
	var request struct {
		Role domain.UserRole `json:"role"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := h.service.SetRole(r.Context(), userID, request.Role)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "authenticated_user_unavailable", "authenticated user is unavailable")
		return
	}
	var input domain.UpdateProfileInput
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := h.service.UpdateProfile(r.Context(), userID, input)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func currentUserID(r *http.Request) (string, bool) {
	token, ok := auth.CurrentToken(r.Context())
	return token.UID, ok
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := httpx.DecodeJSON(w, r, target); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return false
	}
	return true
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, application.ErrUserNotFound) {
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "user was not found")
		return true
	}
	if errors.Is(err, application.ErrValidation) {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return true
	}
	httpx.WriteRequestError(w, r, http.StatusInternalServerError, "user_unavailable", "could not complete the user request")
	return true
}
