// Package response maps customer discovery read models to the stable Flutter
// BusinessModel-compatible HTTP representation shared by customer list routes.
package response

import (
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

func Summaries(items []application.CustomerBusinessSummary) []map[string]any {
	responses := make([]map[string]any, 0, len(items))
	for _, item := range items {
		responses = append(responses, Summary(item))
	}
	return responses
}

func Summary(model application.CustomerBusinessSummary) map[string]any {
	response := base(model.CustomerBusinessBase)
	response["stayDetails"], response["serviceDetails"] = nil, nil
	if model.Stay != nil {
		response["stayDetails"] = staySummary(*model.Stay)
	}
	if model.Service != nil {
		response["serviceDetails"] = serviceSummary(*model.Service)
	}
	return response
}

func Detail(model application.CustomerBusinessDetail) map[string]any {
	response := base(model.CustomerBusinessBase)
	response["stayDetails"], response["serviceDetails"] = nil, nil
	if model.Stay != nil {
		response["stayDetails"] = stayDetail(*model.Stay)
	}
	if model.Service != nil {
		response["serviceDetails"] = serviceDetail(*model.Service)
	}
	return response
}

func RecommendedStay(model application.RecommendedStay) map[string]any {
	response := base(model.CustomerBusinessBase)
	response["stayDetails"] = staySummary(model.Stay)
	return response
}

func base(model application.CustomerBusinessBase) map[string]any {
	return map[string]any{
		"id": model.ID.String(), "ownerId": model.OwnerID, "type": flutterBusinessType(model.Type),
		"name": model.Name, "categoryId": model.CategoryID,
		"location": map[string]any{"city": model.Location.City, "address": model.Location.Address, "latitude": model.Location.Latitude, "longitude": model.Location.Longitude},
		"currency": model.Currency, "shortDescription": model.ShortDescription,
		"logoUrl": model.LogoPath, "coverPhotoUrl": model.CoverPath, "photoUrls": model.PhotoPaths,
		"featuredCollectionIds": model.FeaturedCollectionIDs, "isActive": model.Status == domain.BusinessStatusActive,
		"averageRating": model.AverageRating, "reviewCount": model.ReviewCount,
		"isPromotionActive": model.IsPromotionActive, "createdAt": model.CreatedAt, "updatedAt": model.UpdatedAt,
	}
}

func staySummary(model application.StaySummary) map[string]any {
	return map[string]any{"pricePerNight": model.PricePerNight, "inventoryType": flutterInventoryType(model.InventoryType)}
}

func stayDetail(model application.StayDetail) map[string]any {
	response := staySummary(model.StaySummary)
	extras := make([]map[string]any, 0, len(model.Extras))
	for _, extra := range model.Extras {
		extras = append(extras, map[string]any{"type": extra.Type, "price": extra.Price, "isPerNight": extra.PricingUnit == "per_night", "isPerHour": extra.PricingUnit == "per_hour"})
	}
	rooms := make([]map[string]any, 0, len(model.Rooms))
	for _, room := range model.Rooms {
		rooms = append(rooms, map[string]any{"id": room.ID.String(), "name": room.Name, "maxGuests": room.MaxGuests, "sizeSquareMeters": room.SizeSquareMeters, "pricePerNight": room.PricePerNight, "quantity": room.Quantity, "isActive": room.IsActive})
	}
	response["amenities"], response["extras"], response["rooms"] = model.Amenities, extras, rooms
	return response
}

func serviceSummary(model application.ServiceSummary) map[string]any {
	return map[string]any{"offerings": offeringResponses(model.Offerings)}
}

func serviceDetail(model application.ServiceDetail) map[string]any {
	providers := make([]map[string]any, 0, len(model.Providers))
	for _, provider := range model.Providers {
		slots := make([]map[string]any, 0, len(provider.Availability))
		for _, slot := range provider.Availability {
			slots = append(slots, map[string]any{"id": slot.ID.String(), "weekday": flutterWeekday(slot.Weekday), "startMinutes": slot.StartMinutes, "endMinutes": slot.EndMinutes})
		}
		providers = append(providers, map[string]any{"id": provider.ID.String(), "name": provider.Name, "title": provider.Title, "commissionRate": provider.CommissionRate, "isActive": provider.IsActive, "availabilitySlots": slots})
	}
	return map[string]any{"offerings": offeringResponses(model.Offerings), "availabilitySlots": []any{}, "providers": providers}
}

func offeringResponses(offerings []application.ServiceOffering) []map[string]any {
	items := make([]map[string]any, 0, len(offerings))
	for _, offering := range offerings {
		items = append(items, map[string]any{"id": offering.ID.String(), "name": offering.Name, "description": offering.Description, "durationMinutes": offering.DurationMinutes, "price": offering.Price, "isActive": offering.IsActive})
	}
	return items
}

func flutterBusinessType(value domain.BusinessType) string {
	if value == domain.BusinessTypeStay {
		return "stays"
	}
	return "services"
}

func flutterInventoryType(value string) string {
	if value == "single_unit" {
		return "singleUnit"
	}
	return "multipleUnits"
}

func flutterWeekday(value int16) string {
	return [...]string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}[value]
}
