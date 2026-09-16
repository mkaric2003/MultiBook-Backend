package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mkaric2003/multibook-backend/internal/modules/service_search/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_search/domain"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type Handler struct{ service *application.Service }

func NewHandler(s *application.Service) *Handler { return &Handler{s} }

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	input := domain.Input{
		City: q.Get("city"), CategoryID: q.Get("category_id"), CollectionID: q.Get("collection_id"),
		MinPriceMinor: 0, MaxPriceMinor: 1 << 62, AppointmentDate: q.Get("appointment_date"),
		SortOption: q.Get("sort_option"), Cursor: q.Get("cursor"),
	}
	var err error
	if value := q.Get("min_price_minor"); value != "" {
		input.MinPriceMinor, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			bad(w, r, "min_price_minor must be an integer")
			return
		}
	}
	if value := q.Get("max_price_minor"); value != "" {
		input.MaxPriceMinor, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			bad(w, r, "max_price_minor must be an integer")
			return
		}
	}
	if value := q.Get("start_minutes"); value != "" {
		minute, parseErr := strconv.ParseInt(value, 10, 16)
		if parseErr != nil {
			bad(w, r, "start_minutes must be an integer")
			return
		}
		parsed := int16(minute)
		input.StartMinutes = &parsed
	}
	input.PageSize, _ = strconv.Atoi(q.Get("page_size"))
	page, err := h.service.Search(r.Context(), input)
	if err != nil {
		if errors.Is(err, application.ErrValidation) {
			bad(w, r, strings.TrimPrefix(err.Error(), "validation failed: "))
			return
		}
		httpx.WriteRequestError(w, r, 500, "service_search_unavailable", "could not search services")
		return
	}
	items := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, response(item))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": page.NextCursor})
}

func response(item domain.Item) map[string]any {
	offerings := make([]map[string]any, 0, len(item.Offerings))
	for _, offering := range item.Offerings {
		offerings = append(offerings, map[string]any{"id": offering.ID.String(), "name": offering.Name, "description": offering.Description, "durationMinutes": offering.DurationMinutes, "price": offering.PriceMinor, "isActive": offering.IsActive})
	}
	providers := make([]map[string]any, 0, len(item.Providers))
	for _, provider := range item.Providers {
		slots := make([]map[string]any, 0, len(provider.Availability))
		for _, slot := range provider.Availability {
			slots = append(slots, map[string]any{"id": slot.ID.String(), "weekday": weekday(slot.Weekday), "startMinutes": slot.StartMinutes, "endMinutes": slot.EndMinutes})
		}
		providers = append(providers, map[string]any{"id": provider.ID.String(), "name": provider.Name, "title": provider.Title, "commissionRate": provider.CommissionRate, "isActive": provider.IsActive, "availabilitySlots": slots})
	}
	return map[string]any{
		"id": item.ID.String(), "ownerId": item.OwnerID, "type": "services", "name": item.Name,
		"categoryId": item.CategoryID, "location": map[string]any{"city": item.City, "address": item.Address, "latitude": item.Latitude, "longitude": item.Longitude},
		"currency": item.Currency, "shortDescription": item.ShortDescription, "logoUrl": item.LogoPath,
		"coverPhotoUrl": item.CoverPath, "photoUrls": item.PhotoPaths, "featuredCollectionIds": item.FeaturedCollectionIDs,
		"isActive": true, "averageRating": item.AverageRating, "reviewCount": item.ReviewCount,
		"isPromotionActive": item.IsPromotionActive, "createdAt": item.CreatedAt, "updatedAt": item.UpdatedAt,
		"stayDetails": nil, "serviceDetails": map[string]any{"offerings": offerings, "availabilitySlots": []any{}, "providers": providers},
	}
}

func weekday(value int16) string {
	return []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}[value]
}

func bad(w http.ResponseWriter, r *http.Request, message string) {
	httpx.WriteRequestError(w, r, http.StatusUnprocessableEntity, "validation_error", message)
}
