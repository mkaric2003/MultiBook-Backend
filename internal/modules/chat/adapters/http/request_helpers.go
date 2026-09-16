package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type cursorPayload struct {
	At time.Time `json:"at"`
	ID uuid.UUID `json:"id"`
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

func conversationRequest(w http.ResponseWriter, r *http.Request) (application.Actor, uuid.UUID, bool) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return application.Actor{}, uuid.Nil, false
	}
	conversationID, ok := httpx.PathUUID(w, r, "conversationID", "conversation_id")
	return actor, conversationID, ok
}

func pageInputFromRequest(w http.ResponseWriter, r *http.Request) (*application.Cursor, int, bool) {
	var cursor *application.Cursor
	if value := r.URL.Query().Get("cursor"); value != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			writeInvalidCursor(w, r)
			return nil, 0, false
		}
		var payload cursorPayload
		if err := json.Unmarshal(decoded, &payload); err != nil || payload.At.IsZero() || payload.ID == uuid.Nil {
			writeInvalidCursor(w, r)
			return nil, 0, false
		}
		cursor = &application.Cursor{At: payload.At, ID: payload.ID}
	}
	limit := 0
	if value := r.URL.Query().Get("page_size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "page_size must be an integer")
			return nil, 0, false
		}
		limit = parsed
	}
	return cursor, limit, true
}

func encodeCursor(cursor *application.Cursor) *string {
	if cursor == nil {
		return nil
	}
	payload, _ := json.Marshal(cursorPayload{At: cursor.At, ID: cursor.ID})
	value := base64.RawURLEncoding.EncodeToString(payload)
	return &value
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := httpx.DecodeJSON(w, r, target); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return false
	}
	return true
}

func writeInvalidCursor(w http.ResponseWriter, r *http.Request) {
	httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "cursor is invalid")
}
