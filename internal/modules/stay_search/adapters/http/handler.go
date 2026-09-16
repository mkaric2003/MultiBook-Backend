package http

import (
	"errors"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_search/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_search/domain"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	service *application.Service
}

func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	in := domain.Input{City: q.Get("city"), CheckIn: dateOnly(q.Get("check_in")), CheckOut: dateOnly(q.Get("check_out")), InventoryType: q.Get("inventory_type"), Adults: 1, MinPriceMinor: 0, MaxPriceMinor: 1 << 62, CategoryIDs: q["category_id"], CollectionIDs: q["collection_id"], Amenities: q["amenity"]}
	var err error
	for _, field := range []struct {
		key string
		out *int
	}{{"adults", &in.Adults}, {"children", &in.Children}, {"page_size", &in.PageSize}, {"offset", &in.Offset}} {
		if v := q.Get(field.key); v != "" {
			*field.out, err = strconv.Atoi(v)
			if err != nil {
				bad(w, r, field.key+" must be an integer")
				return
			}
		}
	}
	for _, field := range []struct {
		key string
		out *int64
	}{{"min_price_minor", &in.MinPriceMinor}, {"max_price_minor", &in.MaxPriceMinor}} {
		if v := q.Get(field.key); v != "" {
			*field.out, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				bad(w, r, field.key+" must be an integer")
				return
			}
		}
	}
	if v := q.Get("minimum_rating"); v != "" {
		in.MinimumRating, err = strconv.ParseFloat(v, 64)
		if err != nil {
			bad(w, r, "minimum_rating must be a number")
			return
		}
	}
	items, err := h.service.Search(r.Context(), in)
	if err != nil {
		if errors.Is(err, application.ErrValidation) {
			bad(w, r, strings.TrimPrefix(err.Error(), "validation failed: "))
			return
		}
		log.Printf("stay search failed: %v", err)
		httpx.WriteRequestError(w, r, 500, "stay_search_unavailable", "could not search stays")
		return
	}
	response := make([]map[string]any, 0, len(items))
	for _, x := range items {
		response = append(response, map[string]any{"id": x.ID.String(), "ownerId": x.OwnerID, "type": "stays", "name": x.Name, "categoryId": x.CategoryID, "location": map[string]any{"city": x.City, "address": x.Address, "latitude": x.Latitude, "longitude": x.Longitude}, "currency": x.Currency, "shortDescription": x.ShortDescription, "logoUrl": x.LogoURL, "coverPhotoUrl": x.CoverPhotoURL, "photoUrls": x.PhotoURLs, "featuredCollectionIds": x.FeaturedCollectionIDs, "isActive": true, "averageRating": x.AverageRating, "reviewCount": x.ReviewCount, "isPromotionActive": false, "stayDetails": map[string]any{"pricePerNight": x.PricePerNight, "inventoryType": flutterInventory(x.InventoryType), "amenities": x.Amenities, "rooms": []any{}, "extras": []any{}}})
	}
	next := any(nil)
	if len(items) == in.PageSize {
		next = strconv.Itoa(in.Offset + len(items))
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": response, "nextCursor": next})
}
func dateOnly(v string) string {
	if i := strings.IndexByte(v, 'T'); i >= 0 {
		return v[:i]
	}
	return v
}
func flutterInventory(v string) string {
	if v == "single_unit" {
		return "singleUnit"
	}
	return "multipleUnits"
}
func bad(w http.ResponseWriter, r *http.Request, message string) {
	httpx.WriteRequestError(w, r, 422, "validation_error", message)
}
