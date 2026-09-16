// Package http is the HTTP adapter for authenticated in-app notifications.
package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	offset, limit, ok := pageInputFromRequest(w, r)
	if !ok {
		return
	}
	page, err := h.service.List(r.Context(), userID, offset, limit)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, notificationsPageResponse(page))
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	count, err := h.service.UnreadCount(r.Context(), userID)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, unreadCountResponse(count))
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.MarkRead(r.Context(), userID, chi.URLParam(r, "notificationID"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PutDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var request upsertDeviceRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if handleServiceError(w, r, h.service.UpsertDevice(r.Context(), userID, chi.URLParam(r, "deviceID"), request.Token)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.DeleteDevice(r.Context(), userID, chi.URLParam(r, "deviceID"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
