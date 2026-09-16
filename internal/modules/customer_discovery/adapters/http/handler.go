// Package http exposes customer discovery HTTP endpoints.
package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	discoveryresponse "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/http/response"
	"github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathUUID(w, r, "businessID", "business_id")
	if !ok {
		return
	}
	business, err := h.service.GetActiveDetail(r.Context(), id)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, discoveryresponse.Detail(business))
}

func (h *Handler) ListPopular(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	businessType, ok := businessTypeFromRequest(w, r, query.Get("type"))
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 10
	}
	offset := 0
	if rawOffset := query.Get("offset"); rawOffset != "" {
		var err error
		offset, err = strconv.Atoi(rawOffset)
		if err != nil {
			httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "offset must be an integer")
			return
		}
	}
	items, err := h.service.ListPopular(r.Context(), businessType, query.Get("city"), limit, offset)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": discoveryresponse.Summaries(items), "nextOffset": offset + len(items)})
}

func (h *Handler) ListCities(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCities(r.Context())
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListFeaturedCollections(w http.ResponseWriter, r *http.Request) {
	businessType, ok := businessTypeFromRequest(w, r, r.URL.Query().Get("type"))
	if !ok {
		return
	}
	items, err := h.service.ListFeaturedCollections(r.Context(), businessType)
	if handleServiceError(w, r, err) {
		return
	}
	response := make([]map[string]string, 0, len(items))
	for _, item := range items {
		response = append(response, map[string]string{"id": item.ID, "titleKey": item.TitleKey, "subtitleKey": item.SubtitleKey, "imageUrl": item.ImageURL})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": response})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	businessType, ok := businessTypeFromRequest(w, r, query.Get("type"))
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 20
	}
	items, err := h.service.Search(r.Context(), businessType, query.Get("query"), limit)
	if handleServiceError(w, r, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": discoveryresponse.Summaries(items)})
}

func (h *Handler) ListRecommendedStays(w http.ResponseWriter, r *http.Request) {
	user, ok := usershttp.RequireCurrentUser(w, r)
	if !ok {
		return
	}
	city := ""
	if user.City != nil {
		city = *user.City
	}
	items, err := h.service.ListRecommendedStays(r.Context(), city)
	if handleServiceError(w, r, err) {
		return
	}
	response := make([]map[string]any, 0, len(items))
	for _, item := range items {
		response = append(response, discoveryresponse.RecommendedStay(item))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": response})
}

func businessTypeFromRequest(w http.ResponseWriter, r *http.Request, value string) (domain.BusinessType, bool) {
	switch value {
	case "stays":
		return domain.BusinessTypeStay, true
	case "services":
		return domain.BusinessTypeService, true
	default:
		httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", "type must be stays or services")
		return "", false
	}
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, application.ErrBusinessNotFound) {
		httpx.WriteRequestError(w, r, http.StatusNotFound, "not_found", "business was not found")
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
