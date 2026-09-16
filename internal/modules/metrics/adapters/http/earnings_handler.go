package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type earningsMetricsResponse struct {
	BusinessID              uuid.UUID        `json:"businessId"`
	BusinessType            string           `json:"businessType"`
	Currency                string           `json:"currency"`
	StartDate               string           `json:"startDate"`
	EndDate                 string           `json:"endDate"`
	StaffID                 *uuid.UUID       `json:"staffId,omitempty"`
	RevenueMinor            int64            `json:"revenueMinor"`
	ReservationCount        int              `json:"reservationCount"`
	OnlineRevenueMinor      int64            `json:"onlineRevenueMinor"`
	CashRevenueMinor        int64            `json:"cashRevenueMinor"`
	StaffEarningsMinor      int64            `json:"staffEarningsMinor"`
	DailyRevenueMinor       map[string]int64 `json:"dailyRevenueMinor"`
	DailyReservationCount   map[string]int   `json:"dailyReservationCount"`
	DailyOnlineRevenueMinor map[string]int64 `json:"dailyOnlineRevenueMinor"`
	DailyCashRevenueMinor   map[string]int64 `json:"dailyCashRevenueMinor"`
	DailyStaffEarningsMinor map[string]int64 `json:"dailyStaffEarningsMinor"`
}

func (h *Handler) GetEarningsMetrics(w http.ResponseWriter, r *http.Request) {
	actor, businessID, filter, ok := earningsRequest(w, r)
	if !ok {
		return
	}
	metrics, err := h.service.GetEarningsMetrics(r.Context(), actor, businessID, filter)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, earningsResponse(metrics))
}

func (h *Handler) StreamEarningsMetrics(w http.ResponseWriter, r *http.Request) {
	actor, businessID, filter, ok := earningsRequest(w, r)
	if !ok {
		return
	}
	metrics, updates, unsubscribe, err := h.service.OpenEarningsMetricsStream(r.Context(), actor, businessID, filter)
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
	if err := writeEarningsMetrics(w, flusher, controller, metrics); err != nil {
		return
	}

	heartbeat := time.NewTicker(metricsHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-updates:
			metrics, err = h.service.GetEarningsMetrics(r.Context(), actor, businessID, filter)
			if err != nil || writeEarningsMetrics(w, flusher, controller, metrics) != nil {
				return
			}
		case <-heartbeat.C:
			if err := writeHeartbeat(w, flusher, controller); err != nil {
				return
			}
		}
	}
}

func earningsRequest(w http.ResponseWriter, r *http.Request) (application.Actor, uuid.UUID, application.EarningsFilter, bool) {
	actor, businessID, ok := dashboardRequest(w, r)
	if !ok {
		return application.Actor{}, uuid.Nil, application.EarningsFilter{}, false
	}
	offsetMinutes := 0
	if value := r.URL.Query().Get("utcOffsetMinutes"); value != "" {
		parsedOffset, err := strconv.Atoi(value)
		if err != nil || parsedOffset < -14*60 || parsedOffset > 14*60 {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "utcOffsetMinutes must be an integer between -840 and 840")
			return application.Actor{}, uuid.Nil, application.EarningsFilter{}, false
		}
		offsetMinutes = parsedOffset
	}
	location := time.FixedZone("earnings-client", offsetMinutes*60)
	startDate, startErr := time.ParseInLocation(time.DateOnly, r.URL.Query().Get("startDate"), location)
	endDate, endErr := time.ParseInLocation(time.DateOnly, r.URL.Query().Get("endDate"), location)
	if startErr != nil || endErr != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "startDate and endDate must use YYYY-MM-DD")
		return application.Actor{}, uuid.Nil, application.EarningsFilter{}, false
	}
	filter := application.EarningsFilter{
		StartDate:        startDate,
		EndDate:          endDate,
		UTCOffsetMinutes: offsetMinutes,
	}
	if value := r.URL.Query().Get("staffId"); value != "" {
		staffID, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "staffId must be a UUID")
			return application.Actor{}, uuid.Nil, application.EarningsFilter{}, false
		}
		filter.StaffID = &staffID
	}
	return actor, businessID, filter, true
}

func earningsResponse(metrics application.EarningsMetrics) earningsMetricsResponse {
	return earningsMetricsResponse{
		BusinessID:              metrics.BusinessID,
		BusinessType:            string(metrics.BusinessType),
		Currency:                metrics.Currency,
		StartDate:               metrics.StartDate,
		EndDate:                 metrics.EndDate,
		StaffID:                 metrics.StaffID,
		RevenueMinor:            metrics.RevenueMinor,
		ReservationCount:        metrics.ReservationCount,
		OnlineRevenueMinor:      metrics.OnlineRevenueMinor,
		CashRevenueMinor:        metrics.CashRevenueMinor,
		StaffEarningsMinor:      metrics.StaffEarningsMinor,
		DailyRevenueMinor:       metrics.DailyRevenueMinor,
		DailyReservationCount:   metrics.DailyReservationCount,
		DailyOnlineRevenueMinor: metrics.DailyOnlineRevenueMinor,
		DailyCashRevenueMinor:   metrics.DailyCashRevenueMinor,
		DailyStaffEarningsMinor: metrics.DailyStaffEarningsMinor,
	}
}

func writeEarningsMetrics(w http.ResponseWriter, flusher http.Flusher, controller *http.ResponseController, metrics application.EarningsMetrics) error {
	payload, err := json.Marshal(earningsResponse(metrics))
	if err != nil {
		return err
	}
	if err := setStreamWriteDeadline(controller); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: earnings_metrics\ndata: %s\n\n", payload); err != nil {
		return err
	}
	flusher.Flush()
	return clearStreamWriteDeadline(controller)
}
