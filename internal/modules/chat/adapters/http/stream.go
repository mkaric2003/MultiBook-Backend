package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

const chatHeartbeatInterval = 20 * time.Second

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	updates, unsubscribe, err := h.service.OpenUpdates(actor)
	if handleServiceError(w, r, err) {
		return
	}
	defer unsubscribe()

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "streaming_unavailable", "streaming is unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	if err := writeChatEvent(w, flusher, controller, "chat_sync", map[string]any{}); err != nil {
		return
	}

	heartbeat := time.NewTicker(chatHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case change, open := <-updates:
			if !open {
				return
			}
			if change.ConversationID == nil {
				if writeChatEvent(w, flusher, controller, "chat_sync", map[string]any{}) != nil {
					return
				}
				continue
			}
			visible, err := h.service.CanViewChange(r.Context(), actor, *change.ConversationID)
			if err != nil {
				return
			}
			if visible && writeChatEvent(w, flusher, controller, "chat_changed", map[string]uuid.UUID{"conversationId": *change.ConversationID}) != nil {
				return
			}
		case <-heartbeat.C:
			if writeChatHeartbeat(w, flusher, controller) != nil {
				return
			}
		}
	}
}

func writeChatEvent(w http.ResponseWriter, flusher http.Flusher, controller *http.ResponseController, eventName string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := setChatWriteDeadline(controller); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, payload); err != nil {
		return err
	}
	flusher.Flush()
	return clearChatWriteDeadline(controller)
}

func writeChatHeartbeat(w http.ResponseWriter, flusher http.Flusher, controller *http.ResponseController) error {
	if err := setChatWriteDeadline(controller); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
		return err
	}
	flusher.Flush()
	return clearChatWriteDeadline(controller)
}

func setChatWriteDeadline(controller *http.ResponseController) error {
	err := controller.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}

func clearChatWriteDeadline(controller *http.ResponseController) error {
	err := controller.SetWriteDeadline(time.Time{})
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}
