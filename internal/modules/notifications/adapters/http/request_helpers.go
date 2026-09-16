package http

import (
	"net/http"
	"strconv"

	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

func userIDFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return "", false
	}
	return user.ID, true
}

func pageInputFromRequest(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	query := r.URL.Query()
	offset := 0
	if value := query.Get("cursor"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "cursor must be a non-negative integer")
			return 0, 0, false
		}
		offset = parsed
	}
	limit := 0
	if value := query.Get("page_size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "page_size must be an integer")
			return 0, 0, false
		}
		limit = parsed
	}
	return offset, limit, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := httpx.DecodeJSON(w, r, target); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return false
	}
	return true
}
