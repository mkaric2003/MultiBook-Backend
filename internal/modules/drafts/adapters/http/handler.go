// Package http exposes the authenticated customer's saved drafts.
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	user, ok := customer(w, r)
	if !ok {
		return
	}
	draft, err := h.service.GetBooking(r.Context(), user.ID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bookingResponse(user.ID, draft))
}

func (h *Handler) PutBooking(w http.ResponseWriter, r *http.Request) {
	user, ok := customer(w, r)
	if !ok {
		return
	}
	var request bookingRequest
	if httpx.DecodeJSON(w, r, &request) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	draft, err := h.service.SaveBooking(r.Context(), user.ID, request.input())
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bookingResponse(user.ID, draft))
}

func (h *Handler) DeleteBooking(w http.ResponseWriter, r *http.Request) {
	user, ok := customer(w, r)
	if !ok {
		return
	}
	if writeError(w, r, h.service.DeleteBooking(r.Context(), user.ID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetAppointment(w http.ResponseWriter, r *http.Request) {
	user, ok := customer(w, r)
	if !ok {
		return
	}
	draft, err := h.service.GetAppointment(r.Context(), user.ID)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, appointmentResponse(user.ID, draft))
}

func (h *Handler) PutAppointment(w http.ResponseWriter, r *http.Request) {
	user, ok := customer(w, r)
	if !ok {
		return
	}
	var request appointmentRequest
	if httpx.DecodeJSON(w, r, &request) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	draft, err := h.service.SaveAppointment(r.Context(), user.ID, request.input())
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, appointmentResponse(user.ID, draft))
}

func (h *Handler) DeleteAppointment(w http.ResponseWriter, r *http.Request) {
	user, ok := customer(w, r)
	if !ok {
		return
	}
	if writeError(w, r, h.service.DeleteAppointment(r.Context(), user.ID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type bookingRequest struct {
	BusinessID     string          `json:"businessId"`
	CheckIn        string          `json:"checkIn"`
	CheckOut       string          `json:"checkOut"`
	Adults         int16           `json:"adults"`
	Children       int16           `json:"children"`
	Infants        int16           `json:"infants"`
	RoomTypeID     *string         `json:"roomTypeId"`
	SelectedExtras json.RawMessage `json:"selectedExtras"`
}

func (r bookingRequest) input() domain.BookingInput {
	businessID, _ := uuid.Parse(r.BusinessID)
	var roomTypeID *uuid.UUID
	if r.RoomTypeID != nil {
		if value, err := uuid.Parse(*r.RoomTypeID); err == nil {
			roomTypeID = &value
		}
	}
	return domain.BookingInput{BusinessID: businessID, CheckIn: r.CheckIn, CheckOut: r.CheckOut, Adults: r.Adults, Children: r.Children, Infants: r.Infants, RoomTypeID: roomTypeID, SelectedExtras: r.SelectedExtras}
}

type appointmentRequest struct {
	BusinessID          string          `json:"businessId"`
	SelectedOfferingIDs []string        `json:"selectedOfferingIds"`
	SelectedProviderID  *string         `json:"selectedProviderId"`
	Date                string          `json:"date"`
	StartMinutes        *int16          `json:"startMinutes"`
	SelectedAddOnIDs    json.RawMessage `json:"selectedAddOnIds"`
}

func (r appointmentRequest) input() domain.AppointmentInput {
	businessID, _ := uuid.Parse(r.BusinessID)
	offerings := make([]uuid.UUID, 0, len(r.SelectedOfferingIDs))
	for _, raw := range r.SelectedOfferingIDs {
		if value, err := uuid.Parse(raw); err == nil {
			offerings = append(offerings, value)
		}
	}
	var providerID *uuid.UUID
	if r.SelectedProviderID != nil {
		if value, err := uuid.Parse(*r.SelectedProviderID); err == nil {
			providerID = &value
		}
	}
	return domain.AppointmentInput{BusinessID: businessID, SelectedOfferingIDs: offerings, SelectedProviderID: providerID, AppointmentDate: r.Date, StartMinutes: r.StartMinutes, SelectedAddOnIDs: r.SelectedAddOnIDs}
}

func bookingResponse(customerID string, draft domain.BookingDraft) map[string]any {
	return map[string]any{"id": customerID, "businessId": draft.BusinessID.String(), "businessName": draft.BusinessName, "businessLocation": draft.BusinessLocation, "businessImageUrl": draft.BusinessImageURL, "pricePerNight": draft.PricePerNight, "checkIn": draft.CheckIn, "checkOut": draft.CheckOut, "adults": draft.Adults, "children": draft.Children, "infants": draft.Infants, "roomTypeId": draft.RoomTypeID, "selectedExtras": draft.SelectedExtras, "updatedAt": draft.UpdatedAt}
}
func appointmentResponse(customerID string, draft domain.AppointmentDraft) map[string]any {
	offerings := make([]string, len(draft.SelectedOfferingIDs))
	for i, id := range draft.SelectedOfferingIDs {
		offerings[i] = id.String()
	}
	var providerID *string
	if draft.SelectedProviderID != nil {
		value := draft.SelectedProviderID.String()
		providerID = &value
	}
	return map[string]any{"id": customerID, "businessId": draft.BusinessID.String(), "businessName": draft.BusinessName, "businessImageUrl": draft.BusinessImageURL, "selectedOfferingIds": offerings, "selectedProviderId": providerID, "selectedProviderName": draft.SelectedProviderName, "date": draft.AppointmentDate, "startMinutes": draft.StartMinutes, "selectedAddOnIds": draft.SelectedAddOnIDs, "updatedAt": draft.UpdatedAt}
}

func customer(w http.ResponseWriter, r *http.Request) (usersdomain.User, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return usersdomain.User{}, false
	}
	if user.Role == nil || string(*user.Role) != "customer" {
		httpx.WriteRequestError(w, r, 403, "forbidden", "a customer role is required")
		return usersdomain.User{}, false
	}
	return user, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, 404, "not_found", "draft or its business was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	default:
		httpx.WriteRequestError(w, r, 500, "draft_unavailable", "could not access draft")
	}
	return true
}
