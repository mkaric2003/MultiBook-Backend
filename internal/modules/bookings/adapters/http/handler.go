package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	var input domain.CreateInput
	if httpx.DecodeJSON(w, r, &input) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	booking, err := h.service.Create(r.Context(), user.ID, role, businessID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, booking)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := actor(w, r)
	if !ok {
		return
	}
	input := domain.ListInput{Status: r.URL.Query().Get("status")}
	query := r.URL.Query()
	if value := query.Get("business_id"); value != "" {
		businessID, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "business_id must be a UUID")
			return
		}
		input.BusinessID = &businessID
	}
	var err error
	if value := query.Get("cursor"); value != "" {
		input.Offset, err = strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "cursor must be an integer")
			return
		}
	}
	if value := query.Get("page_size"); value != "" {
		input.Limit, err = strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "page_size must be an integer")
			return
		}
	}
	page, err := h.service.List(r.Context(), userID, role, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": page.Items, "nextCursor": page.NextCursor})
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := actor(w, r)
	if !ok {
		return
	}
	bookingID, ok := httpx.PathUUID(w, r, "bookingID", "booking_id")
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if httpx.DecodeJSON(w, r, &input) != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return
	}
	booking, err := h.service.UpdateStatus(r.Context(), userID, role, bookingID, input.Status)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, booking)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := actor(w, r)
	if !ok {
		return
	}
	bookingID, ok := httpx.PathUUID(w, r, "bookingID", "booking_id")
	if !ok {
		return
	}
	booking, err := h.service.Cancel(r.Context(), userID, role, bookingID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, booking)
}

func (h *Handler) Availability(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := actor(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	var roomTypeID *uuid.UUID
	if value := r.URL.Query().Get("room_type_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "room_type_id must be a UUID")
			return
		}
		roomTypeID = &parsed
	}
	ranges, err := h.service.Availability(r.Context(), userID, role, businessID, roomTypeID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"unavailableRanges": ranges})
}

func actor(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return "", "", false
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	return user.ID, role, true
}
func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, 404, "not_found", "stay or room type was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, 403, "forbidden", "a customer role is required")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	case errors.Is(err, application.ErrConflict):
		httpx.WriteRequestError(w, r, 409, "stay_unavailable", "the selected dates are no longer available")
	default:
		httpx.WriteRequestError(w, r, 500, "booking_unavailable", "could not create booking")
	}
	return true
}
