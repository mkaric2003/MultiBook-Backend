package http

import (
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

func ownedSummaryResponse(model application.OwnedSummary) map[string]any {
	response := businessReadBaseResponse(model.BusinessReadBase)
	response["stayDetails"] = nil
	if model.Stay != nil {
		response["stayDetails"] = staySummaryResponse(*model.Stay)
	}
	return response
}

func ownedDetailResponse(model application.OwnedDetail) map[string]any {
	response := businessReadBaseResponse(model.BusinessReadBase)
	response["stayDetails"], response["serviceDetails"] = nil, nil
	if model.Stay != nil {
		response["stayDetails"] = stayDetailReadResponse(*model.Stay)
	}
	if model.Service != nil {
		response["serviceDetails"] = serviceDetailReadResponse(*model.Service)
	}
	return response
}

func businessReadBaseResponse(model application.BusinessReadBase) map[string]any {
	return map[string]any{
		"id":                    model.ID.String(),
		"ownerId":               model.OwnerID,
		"type":                  flutterBusinessType(model.Type),
		"name":                  model.Name,
		"categoryId":            model.CategoryID,
		"location":              map[string]any{"city": model.Location.City, "address": model.Location.Address, "latitude": model.Location.Latitude, "longitude": model.Location.Longitude},
		"currency":              model.Currency,
		"shortDescription":      model.ShortDescription,
		"logoUrl":               model.LogoPath,
		"coverPhotoUrl":         model.CoverPath,
		"photoUrls":             model.PhotoPaths,
		"featuredCollectionIds": model.FeaturedCollectionIDs,
		"isActive":              model.Status == domain.BusinessStatusActive,
		"averageRating":         model.AverageRating,
		"reviewCount":           model.ReviewCount,
		"isPromotionActive":     model.IsPromotionActive,
		"createdAt":             model.CreatedAt,
		"updatedAt":             model.UpdatedAt,
	}
}

func staySummaryResponse(model application.StaySummaryRead) map[string]any {
	return map[string]any{"pricePerNight": model.PricePerNight, "inventoryType": flutterInventoryType(model.InventoryType)}
}

func stayDetailReadResponse(model application.StayDetailRead) map[string]any {
	response := staySummaryResponse(model.StaySummaryRead)
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

func serviceDetailReadResponse(model application.ServiceDetailRead) map[string]any {
	offerings := make([]map[string]any, 0, len(model.Offerings))
	for _, offering := range model.Offerings {
		offerings = append(offerings, map[string]any{"id": offering.ID.String(), "name": offering.Name, "description": offering.Description, "durationMinutes": offering.DurationMinutes, "price": offering.Price, "isActive": offering.IsActive})
	}
	providers := make([]map[string]any, 0, len(model.Providers))
	for _, provider := range model.Providers {
		slots := make([]map[string]any, 0, len(provider.Availability))
		for _, slot := range provider.Availability {
			slots = append(slots, map[string]any{"id": slot.ID.String(), "weekday": flutterWeekday(slot.Weekday), "startMinutes": slot.StartMinutes, "endMinutes": slot.EndMinutes})
		}
		providers = append(providers, map[string]any{"id": provider.ID.String(), "name": provider.Name, "title": provider.Title, "commissionRate": provider.CommissionRate, "isActive": provider.IsActive, "availabilitySlots": slots})
	}
	return map[string]any{"offerings": offerings, "availabilitySlots": []any{}, "providers": providers}
}

// businessModelResponse deliberately matches the existing Flutter BusinessModel
// mapper. Create returns the complete aggregate, so the client can decode the
// response directly with BusinessModel.fromMap without merging it with a draft.
func businessModelResponse(b domain.Business, input domain.CreateInput) map[string]any {
	response := map[string]any{
		"id":                    b.ID.String(),
		"ownerId":               b.OwnerID,
		"type":                  flutterBusinessType(b.Type),
		"name":                  b.Name,
		"categoryId":            b.CategoryID,
		"location":              map[string]any{"city": input.Location.City, "address": input.Location.Address, "latitude": input.Location.Latitude, "longitude": input.Location.Longitude},
		"currency":              b.Currency,
		"shortDescription":      b.ShortDescription,
		"featuredCollectionIds": input.FeaturedCollectionIDs,
		"isActive":              b.Status == domain.BusinessStatusActive,
		"averageRating":         b.AverageRating,
		"reviewCount":           b.ReviewCount,
		"isPromotionActive":     b.IsPromotionActive,
		"createdAt":             b.CreatedAt,
		"updatedAt":             b.UpdatedAt,
		"photoUrls":             []string{},
	}
	for _, media := range input.Media {
		switch media.MediaType {
		case "logo":
			response["logoUrl"] = media.StoragePath
		case "cover":
			response["coverPhotoUrl"] = media.StoragePath
		case "gallery":
			response["photoUrls"] = append(response["photoUrls"].([]string), media.StoragePath)
		}
	}
	if input.Stay != nil {
		response["stayDetails"] = stayDetailsResponse(*input.Stay)
	}
	if input.Service != nil {
		response["serviceDetails"] = serviceDetailsResponse(*input.Service)
	}
	return response
}

func flutterBusinessType(value domain.BusinessType) string {
	if value == domain.BusinessTypeStay {
		return "stays"
	}
	return "services"
}

func stayDetailsResponse(stay domain.Stay) map[string]any {
	rooms := make([]map[string]any, 0, len(stay.UnitTypes))
	for _, unit := range stay.UnitTypes {
		active := true
		if unit.IsActive != nil {
			active = *unit.IsActive
		}
		rooms = append(rooms, map[string]any{"id": unit.ID.String(), "name": unit.Name, "maxGuests": unit.MaxGuests, "sizeSquareMeters": unit.SizeSquareMeters, "pricePerNight": unit.PricePerNightMinor, "quantity": unit.Quantity, "isActive": active})
	}
	extras := make([]map[string]any, 0, len(stay.Extras))
	for _, extra := range stay.Extras {
		extras = append(extras, map[string]any{"type": extra.Type, "price": extra.PriceMinor, "isPerNight": extra.PricingUnit == "per_night", "isPerHour": extra.PricingUnit == "per_hour"})
	}
	return map[string]any{"pricePerNight": stay.BasePriceMinor, "inventoryType": flutterInventoryType(stay.InventoryType), "amenities": stay.Amenities, "rooms": rooms, "extras": extras}
}

func flutterInventoryType(value string) string {
	if value == "single_unit" {
		return "singleUnit"
	}
	return "multipleUnits"
}

func serviceDetailsResponse(service domain.Service) map[string]any {
	offerings := make([]map[string]any, 0, len(service.Offerings))
	for _, offering := range service.Offerings {
		active := true
		if offering.IsActive != nil {
			active = *offering.IsActive
		}
		offerings = append(offerings, map[string]any{"id": offering.ID.String(), "name": offering.Name, "description": offering.Description, "durationMinutes": offering.DurationMinutes, "price": offering.PriceMinor, "isActive": active})
	}
	providers := make([]map[string]any, 0, len(service.Staff))
	for _, staff := range service.Staff {
		active, commission := true, float64(100)
		if staff.IsActive != nil {
			active = *staff.IsActive
		}
		if staff.CommissionRate != nil {
			commission = *staff.CommissionRate
		}
		slots := make([]map[string]any, 0, len(staff.WeeklyAvailability))
		for _, slot := range staff.WeeklyAvailability {
			slots = append(slots, map[string]any{"id": slot.ID.String(), "weekday": flutterWeekday(slot.Weekday), "startMinutes": slot.StartMinutes, "endMinutes": slot.EndMinutes})
		}
		providers = append(providers, map[string]any{"id": staff.ID.String(), "name": staff.Name, "title": staff.Title, "commissionRate": commission, "availabilitySlots": slots, "isActive": active})
	}
	return map[string]any{"offerings": offerings, "availabilitySlots": []any{}, "providers": providers}
}

func flutterWeekday(value int16) string {
	return []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}[value]
}
