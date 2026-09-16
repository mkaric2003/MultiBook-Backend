package http

import (
	"errors"
	"net/http"

	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "conversation was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this conversation")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "chat input is invalid")
	case errors.Is(err, application.ErrConflict):
		httpx.WriteRequestError(w, r, http.StatusConflict, "message_id_conflict", "message id was already used with different content")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "chat_unavailable", "could not complete the chat request")
	}
	return true
}
