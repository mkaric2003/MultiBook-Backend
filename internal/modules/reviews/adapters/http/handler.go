// Package http exposes Reviews HTTP endpoints.
package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

type createRequest struct {
	SourceID string            `json:"sourceId"`
	Type     domain.SourceType `json:"type"`
	Rating   int16             `json:"rating"`
	Comment  *string           `json:"comment"`
}

type reviewResponse struct {
	ID                uuid.UUID `json:"id"`
	CustomerName      string    `json:"customerName"`
	CustomerAvatarURL *string   `json:"customerAvatarUrl"`
	Rating            int16     `json:"rating"`
	Comment           *string   `json:"comment"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	var body createRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteRequestError(w, r, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	sourceID, err := uuid.Parse(body.SourceID)
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "sourceId must be a UUID")
		return
	}
	review, err := h.service.Create(r.Context(), actor, businessID, domain.CreateInput{
		SourceID: sourceID, Type: body.Type, Rating: body.Rating, Comment: body.Comment,
	})
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response(review))
}

func (h *Handler) HasReview(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	hasReview, err := h.service.HasReview(r.Context(), actor, businessID)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"hasReview": hasReview})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(w, r)
	if !ok {
		return
	}
	businessID, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	limit, err := optionalInt(r, "limit")
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "limit must be an integer")
		return
	}
	offset, err := optionalInt(r, "offset")
	if err != nil {
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "offset must be an integer")
		return
	}
	page, err := h.service.List(r.Context(), actor, businessID, limit, offset)
	if handleServiceError(w, r, err) {
		return
	}
	items := make([]reviewResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = response(item)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "nextOffset": page.NextOffset})
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
	return application.Actor{ID: user.ID, Role: role}, true
}

func response(review domain.Review) reviewResponse {
	return reviewResponse{ID: review.ID, CustomerName: review.CustomerName, CustomerAvatarURL: review.CustomerAvatarPath, Rating: review.Rating, Comment: review.Comment}
}

func optionalInt(r *http.Request, name string) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "review source or business was not found")
	case errors.Is(err, application.ErrForbidden):
		httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this action")
	case errors.Is(err, application.ErrValidation):
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "review input is invalid")
	case errors.Is(err, application.ErrAlreadyReviewed):
		httpx.WriteRequestError(w, r, http.StatusConflict, "already_reviewed", "you have already reviewed this business")
	case errors.Is(err, application.ErrSourceNotFinished):
		httpx.WriteRequestError(w, r, http.StatusConflict, "source_not_finished", "the booking or appointment is not finished")
	default:
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "reviews_unavailable", "could not complete the Reviews request")
	}
	return true
}
