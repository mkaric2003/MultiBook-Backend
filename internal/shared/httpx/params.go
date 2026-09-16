package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PathUUID parses a UUID path parameter and writes the standard validation
// response when it is missing or malformed.
func PathUUID(w http.ResponseWriter, r *http.Request, paramName, fieldName string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, paramName))
	if err != nil {
		WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", fieldName+" must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}
