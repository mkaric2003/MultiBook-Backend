// Package http exposes provider Metrics HTTP endpoints.
package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

const metricsHeartbeatInterval = 20 * time.Second

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

type dashboardMetricsResponse struct {
	BusinessID             uuid.UUID              `json:"businessId"`
	BusinessType           string                 `json:"businessType"`
	Currency               string                 `json:"currency"`
	ActiveReservationCount int                    `json:"activeReservationCount"`
	CurrentMonth           dashboardMonthResponse `json:"currentMonth"`
}

type dashboardMonthResponse struct {
	MonthKey              string           `json:"monthKey"`
	RevenueMinor          int64            `json:"revenueMinor"`
	ReservationCount      int              `json:"reservationCount"`
	DailyRevenueMinor     map[string]int64 `json:"dailyRevenueMinor"`
	DailyReservationCount map[string]int   `json:"dailyReservationCount"`
}

func (h *Handler) GetDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := dashboardRequest(w, r)
	if !ok {
		return
	}
	metrics, err := h.service.GetDashboardMetrics(r.Context(), actor, businessID)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dashboardResponse(metrics))
}

// StreamDashboardMetrics keeps an SSE response open and emits initial metrics
// followed by fresh source-of-truth metrics after every committed
// reservation change for this business.
func (h *Handler) StreamDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := dashboardRequest(w, r)
	if !ok {
		return
	}
	metrics, updates, unsubscribe, err := h.service.OpenDashboardMetricsStream(r.Context(), actor, businessID)
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
	if err := writeDashboardMetrics(w, flusher, controller, metrics); err != nil {
		return
	}

	heartbeat := time.NewTicker(metricsHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-updates:
			metrics, err = h.service.GetDashboardMetrics(r.Context(), actor, businessID)
			if err != nil || writeDashboardMetrics(w, flusher, controller, metrics) != nil {
				return
			}
		case <-heartbeat.C:
			if err := writeHeartbeat(w, flusher, controller); err != nil {
				return
			}
		}
	}
}

func dashboardRequest(w http.ResponseWriter, r *http.Request) (application.Actor, uuid.UUID, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return application.Actor{}, uuid.Nil, false
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return application.Actor{}, uuid.Nil, false
	}
	return application.Actor{ID: user.ID, Role: role}, businessID, true
}

func dashboardResponse(metrics application.DashboardMetrics) dashboardMetricsResponse {
	return dashboardMetricsResponse{
		BusinessID:             metrics.BusinessID,
		BusinessType:           string(metrics.BusinessType),
		Currency:               metrics.Currency,
		ActiveReservationCount: metrics.ActiveReservationCount,
		CurrentMonth: dashboardMonthResponse{
			MonthKey:              metrics.CurrentMonth.MonthKey,
			RevenueMinor:          metrics.CurrentMonth.RevenueMinor,
			ReservationCount:      metrics.CurrentMonth.ReservationCount,
			DailyRevenueMinor:     metrics.CurrentMonth.DailyRevenueMinor,
			DailyReservationCount: metrics.CurrentMonth.DailyReservationCount,
		},
	}
}

func writeDashboardMetrics(w http.ResponseWriter, flusher http.Flusher, controller *http.ResponseController, metrics application.DashboardMetrics) error {
	payload, err := json.Marshal(dashboardResponse(metrics))
	if err != nil {
		return err
	}
	if err := setStreamWriteDeadline(controller); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: dashboard_metrics\ndata: %s\n\n", payload); err != nil {
		return err
	}
	flusher.Flush()
	return clearStreamWriteDeadline(controller)
}

func setStreamWriteDeadline(controller *http.ResponseController) error {
	err := controller.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}

func clearStreamWriteDeadline(controller *http.ResponseController) error {
	err := controller.SetWriteDeadline(time.Time{})
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}

func writeHeartbeat(w http.ResponseWriter, flusher http.Flusher, controller *http.ResponseController) error {
	if err := setStreamWriteDeadline(controller); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
		return err
	}
	flusher.Flush()
	return clearStreamWriteDeadline(controller)
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrBusinessNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "business_not_found", "business was not found")
	case errors.Is(err, application.ErrStaffNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "staff_not_found", "staff member was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this business")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "business id is invalid")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "metrics_unavailable", "could not load metrics")
	}
	return true
}
