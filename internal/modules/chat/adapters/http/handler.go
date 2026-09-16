// Package http exposes authenticated chat HTTP endpoints.
package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetOrCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		BusinessID string `json:"businessId"`
		CustomerID string `json:"customerId"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	businessID, err := uuid.Parse(body.BusinessID)
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "businessId must be a UUID")
		return
	}
	conversation, err := h.service.GetOrCreate(r.Context(), actor, businessID, body.CustomerID)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationResponse(conversation))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit, ok := pageInputFromRequest(w, r)
	if !ok {
		return
	}
	page, err := h.service.List(r.Context(), actor, cursor, limit)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationPageResponse(page))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actor, conversationID, ok := conversationRequest(w, r)
	if !ok {
		return
	}
	conversation, err := h.service.Get(r.Context(), actor, conversationID)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, conversationResponse(conversation))
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	actor, conversationID, ok := conversationRequest(w, r)
	if !ok {
		return
	}
	cursor, limit, ok := pageInputFromRequest(w, r)
	if !ok {
		return
	}
	page, err := h.service.ListMessages(r.Context(), actor, conversationID, cursor, limit)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, messagePageResponse(page))
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	actor, conversationID, ok := conversationRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	messageID, err := uuid.Parse(body.ID)
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "id must be a UUID")
		return
	}
	message, err := h.service.SendMessage(r.Context(), actor, conversationID, messageID, body.Text)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, messageResponse(message))
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	actor, conversationID, ok := conversationRequest(w, r)
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.MarkRead(r.Context(), actor, conversationID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetTyping(w http.ResponseWriter, r *http.Request) {
	h.setParticipantFlag(w, r, h.service.SetTyping)
}

func (h *Handler) SetPresence(w http.ResponseWriter, r *http.Request) {
	h.setParticipantFlag(w, r, h.service.SetActive)
}

func (h *Handler) setParticipantFlag(w http.ResponseWriter, r *http.Request, update func(context.Context, application.Actor, uuid.UUID, bool) error) {
	actor, conversationID, ok := conversationRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if handleServiceError(w, r, update(r.Context(), actor, conversationID, body.Active)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	count, err := h.service.UnreadCount(r.Context(), actor)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}
