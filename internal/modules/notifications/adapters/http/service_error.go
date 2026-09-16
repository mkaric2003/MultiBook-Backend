package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, application.ErrNotFound) {
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "notification was not found")
		return true
	}
	if errors.Is(err, application.ErrValidation) {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
		return true
	}
	httpx.WriteRequestError(w, r, http.StatusInternalServerError, "notifications_unavailable", "could not access notifications")
	return true
}
