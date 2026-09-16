// Package http exposes promotion REST endpoints.
package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

type createRequest struct {
	Name          string      `json:"name"`
	Type          domain.Type `json:"type"`
	Value         int64       `json:"value"`
	StartsAt      string      `json:"startsAt"`
	EndsAt        string      `json:"endsAt"`
	Code          *string     `json:"code"`
	MinimumAmount int64       `json:"minimumAmount"`
	MinimumNights int         `json:"minimumNights"`
	UsageLimit    *int        `json:"usageLimit"`
}

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	items, err := handler.service.List(r.Context(), actor, businessID)
	if writeError(w, r, err) {
		return
	}
	responses := make([]promotionResponse, len(items))
	for index, promotion := range items {
		responses[index] = response(promotion)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responses})
}

func (handler *Handler) Active(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	var code *string
	if value, exists := r.URL.Query()["promo_code"]; exists && len(value) > 0 {
		code = &value[0]
	}
	promotion, err := handler.service.Active(r.Context(), actor, businessID, code)
	if writeError(w, r, err) {
		return
	}
	if promotion == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"promotion": nil})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"promotion": response(*promotion)})
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	var request createRequest
	if httpx.DecodeJSON(w, r, &request) != nil {
		httpx.WriteRequestError(w, r, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	startsAt, err := time.Parse(time.RFC3339, request.StartsAt)
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "startsAt must be an ISO-8601 timestamp")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, request.EndsAt)
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "endsAt must be an ISO-8601 timestamp")
		return
	}
	promotion, err := handler.service.Create(r.Context(), actor, businessID, domain.CreateInput{
		Name: request.Name, Type: request.Type, Value: request.Value, StartsAt: startsAt, EndsAt: endsAt,
		Code: request.Code, MinimumAmount: request.MinimumAmount, MinimumNights: request.MinimumNights, UsageLimit: request.UsageLimit,
	})
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response(promotion))
}

func (handler *Handler) SetActive(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	promotionID, ok := httpx.PathUUID(w, r, "promotionID", "promotion_id")
	if !ok {
		return
	}
	var request struct {
		IsActive bool `json:"isActive"`
	}
	if httpx.DecodeJSON(w, r, &request) != nil {
		httpx.WriteRequestError(w, r, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	promotion, err := handler.service.SetActive(r.Context(), actor, businessID, promotionID, request.IsActive)
	if writeError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response(promotion))
}

func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, businessID, ok := requestContext(w, r)
	if !ok {
		return
	}
	promotionID, ok := httpx.PathUUID(w, r, "promotionID", "promotion_id")
	if !ok {
		return
	}
	if writeError(w, r, handler.service.Delete(r.Context(), actor, businessID, promotionID)) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func requestContext(w http.ResponseWriter, r *http.Request) (application.Actor, uuid.UUID, bool) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return application.Actor{}, uuid.Nil, false
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return application.Actor{}, uuid.Nil, false
	}
	role := ""
	if user.Role != nil {
		role = string(*user.Role)
	}
	return application.Actor{ID: user.ID, Role: role}, businessID, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "promotion access is forbidden")
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "promotion or business was not found")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "promotion input is invalid")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "promotions_unavailable", "could not complete the promotion request")
	}
	return true
}
