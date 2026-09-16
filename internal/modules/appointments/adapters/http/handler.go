package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/domain"
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
	appointment, err := h.service.Create(r.Context(), user.ID, role, businessID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, appointmentResponse(appointment))
}

func appointmentResponse(appointment domain.Appointment) map[string]any {
	offerings := make([]map[string]any, 0, len(appointment.Offerings))
	for _, offering := range appointment.Offerings {
		offerings = append(offerings, map[string]any{
			"id": offering.ID.String(), "name": offering.Name,
			"durationMinutes": offering.DurationMinutes, "price": offering.PriceMinor,
		})
	}
	serviceIDs := make([]string, 0, len(offerings))
	serviceNames := make([]string, 0, len(offerings))
	for _, offering := range offerings {
		serviceIDs = append(serviceIDs, offering["id"].(string))
		serviceNames = append(serviceNames, offering["name"].(string))
	}
	return map[string]any{
		"id": appointment.ID.String(), "businessId": appointment.BusinessID.String(),
		"businessOwnerId": appointment.BusinessOwnerID, "businessImageUrl": appointment.BusinessImageURL,
		"customerId": appointment.CustomerID, "customerName": appointment.CustomerName,
		"customerEmail": appointment.CustomerEmail, "customerPhone": appointment.CustomerPhone,
		"providerId":   appointment.StaffID.String(),
		"businessName": appointment.BusinessName, "providerName": appointment.ProviderName,
		"date": appointment.AppointmentDate, "startMinutes": appointment.StartMinutes,
		"endMinutes": appointment.EndMinutes, "serviceCost": appointment.ServiceCostMinor,
		"originalServiceCost": appointment.OriginalServiceCostMinor, "discountAmount": appointment.DiscountMinor,
		"serviceFee": appointment.ServiceFeeMinor, "taxes": appointment.TaxesMinor,
		"total": appointment.TotalMinor, "providerCommissionRate": appointment.ProviderCommissionRate,
		"providerEarnings": appointment.ProviderEarningsMinor, "currency": appointment.Currency,
		"paymentStatus": appointment.PaymentStatus, "paymentMethod": appointment.PaymentMethod,
		"confirmationCode": appointment.ConfirmationCode, "status": appointment.Status,
		"rescheduleCount": appointment.CustomerRescheduleCount, "offerings": offerings,
		"serviceIds": serviceIDs, "serviceNames": serviceNames, "addOnsCost": 0,
	}
}
func (h *Handler) AvailableSlots(w http.ResponseWriter, r *http.Request) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	staffID, ok := httpx.PathUUID(w, r, "staffID", "staff_id")
	if !ok {
		return
	}
	input := domain.AvailabilityInput{AppointmentDate: r.URL.Query().Get("appointment_date")}
	for _, value := range r.URL.Query()["offering_id"] {
		id, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteRequestError(w, r, 422, "validation_error", "offering_id must be a UUID")
			return
		}
		input.OfferingIDs = append(input.OfferingIDs, id)
	}
	result, err := h.service.Availability(r.Context(), user.ID, businessID, staffID, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	input := domain.ListInput{Status: r.URL.Query().Get("status")}
	if value := r.URL.Query().Get("business_id"); value != "" {
		businessID, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteRequestError(w, r, 422, "validation_error", "business_id must be a UUID")
			return
		}
		input.BusinessID = businessID
	}
	if value := r.URL.Query().Get("cursor"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, 422, "validation_error", "cursor must be an integer")
			return
		}
		input.Offset = offset
	}
	if value := r.URL.Query().Get("page_size"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteRequestError(w, r, 422, "validation_error", "page_size must be an integer")
			return
		}
		input.Limit = limit
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	page, err := h.service.List(r.Context(), user.ID, role, input)
	if writeError(w, r, err) {
		return
	}
	response := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		response = append(response, appointmentResponse(item))
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": response, "nextCursor": page.NextCursor})
}
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "appointmentID", "appointment_id")
	if !ok {
		return
	}
	var input struct {
		Status domain.Status `json:"status"`
	}
	if httpx.DecodeJSON(w, r, &input) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	item, err := h.service.Status(r.Context(), user.ID, role, id, input.Status)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, 200, appointmentResponse(item))
}
func (h *Handler) Reschedule(w http.ResponseWriter, r *http.Request) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "appointmentID", "appointment_id")
	if !ok {
		return
	}
	var input domain.RescheduleInput
	if httpx.DecodeJSON(w, r, &input) != nil {
		httpx.WriteRequestError(w, r, 422, "validation_error", "request body is invalid")
		return
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	item, err := h.service.Reschedule(r.Context(), user.ID, role, id, input)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, 200, appointmentResponse(item))
}
func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, 404, "not_found", "service business, staff member, or offering was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, 403, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, 422, "validation_error", strings.TrimPrefix(err.Error(), "validation failed: "))
	case errors.Is(err, application.ErrConflict):
		httpx.WriteRequestError(w, r, 409, "appointment_unavailable", "the selected time is no longer available")
	default:
		httpx.WriteRequestError(w, r, 500, "appointment_unavailable", "could not create appointment")
	}
	return true
}
