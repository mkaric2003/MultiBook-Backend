// Package http is the HTTP adapter for the businesses module.
package http

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	var request createBusinessRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	input, err := request.toDomain()
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	created, err := h.service.Create(r.Context(), actor, input)
	if err != nil {
		log.Printf("create business failed: %v", err)
	}
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, businessModelResponse(created.Business, created.Input))
}

func (h *Handler) GetOwned(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	business, err := h.service.GetOwned(r.Context(), actor, id)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ownedDetailResponse(business))
}

func (h *Handler) ListOwned(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	businesses, err := h.service.ListOwned(r.Context(), actor)
	if handleServiceError(w, r, err) {
		return
	}
	items := make([]map[string]any, 0, len(businesses))
	for _, business := range businesses {
		items = append(items, ownedSummaryResponse(business))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	var input domain.UpdateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	business, err := h.service.Update(r.Context(), actor, id, input)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, business)
}

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	var request createBusinessRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	input, err := request.toDomain()
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	created, err := h.service.Replace(r.Context(), actor, id, input)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, businessModelResponse(created.Business, created.Input))
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	if handleServiceError(w, r, h.service.Archive(r.Context(), actor, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Select(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	var request struct {
		BusinessID uuid.UUID `json:"business_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.BusinessID == uuid.Nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "business_id is required")
		return
	}
	if handleServiceError(w, r, h.service.Select(r.Context(), actor, request.BusinessID)) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"selected_business_id": request.BusinessID.String()})
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
	return application.Actor{ID: user.ID, Role: role, Currency: user.BusinessCurrency}, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := httpx.DecodeJSON(w, r, target); err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "request body is invalid")
		return false
	}
	return true
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, application.ErrBusinessNotFound) {
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "business was not found")
		return true
	}
	if errors.Is(err, application.ErrForbidden) {
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this action")
		return true
	}
	if errors.Is(err, application.ErrValidation) {
		message := strings.TrimPrefix(err.Error(), "validation failed: ")
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", message)
		return true
	}
	httpx.WriteRequestError(w, r, http.StatusInternalServerError, "business_unavailable", "could not complete the business request")
	return true
}
