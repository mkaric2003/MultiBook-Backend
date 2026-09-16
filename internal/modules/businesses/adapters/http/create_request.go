package http

import (
	"fmt"

	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// createBusinessRequest has the exact JSON shape emitted by Flutter's existing
// BusinessModel.toMap(). Database and domain naming remain internal.
type createBusinessRequest struct {
	// These fields are emitted by BusinessModel.toMap(). They are server-owned
	// and ignored during creation, but accepted by the strict JSON decoder.
	ID                    string                `json:"id"`
	OwnerID               string                `json:"ownerId"`
	Type                  string                `json:"type"`
	Name                  string                `json:"name"`
	CategoryID            string                `json:"categoryId"`
	Location              createLocationRequest `json:"location"`
	ShortDescription      *string               `json:"shortDescription"`
	LogoURL               *string               `json:"logoUrl"`
	CoverPhotoURL         *string               `json:"coverPhotoUrl"`
	PhotoURLs             []string              `json:"photoUrls"`
	FeaturedCollectionIDs []string              `json:"featuredCollectionIds"`
	StayDetails           *createStayRequest    `json:"stayDetails"`
	ServiceDetails        *createServiceRequest `json:"serviceDetails"`
	Currency              string                `json:"currency"`
	IsActive              bool                  `json:"isActive"`
	AverageRating         float64               `json:"averageRating"`
	ReviewCount           int                   `json:"reviewCount"`
	IsPromotionActive     bool                  `json:"isPromotionActive"`
	CreatedAt             any                   `json:"createdAt"`
	UpdatedAt             any                   `json:"updatedAt"`
}

type createLocationRequest struct {
	City      string  `json:"city"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type createStayRequest struct {
	PricePerNight *int64               `json:"pricePerNight"`
	InventoryType string               `json:"inventoryType"`
	Amenities     []string             `json:"amenities"`
	Rooms         []createRoomRequest  `json:"rooms"`
	Extras        []createExtraRequest `json:"extras"`
}
type createRoomRequest struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	MaxGuests        int16  `json:"maxGuests"`
	SizeSquareMeters int    `json:"sizeSquareMeters"`
	PricePerNight    int64  `json:"pricePerNight"`
	Quantity         int16  `json:"quantity"`
	IsActive         *bool  `json:"isActive"`
}
type createExtraRequest struct {
	Type       string `json:"type"`
	Price      int64  `json:"price"`
	IsPerNight bool   `json:"isPerNight"`
	IsPerHour  bool   `json:"isPerHour"`
}
type createServiceRequest struct {
	Offerings         []createOfferingRequest     `json:"offerings"`
	AvailabilitySlots []createAvailabilityRequest `json:"availabilitySlots"`
	Provider          *createStaffRequest         `json:"provider"`
	Providers         []createStaffRequest        `json:"providers"`
}
type createOfferingRequest struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	DurationMinutes int16   `json:"durationMinutes"`
	Price           int64   `json:"price"`
	IsActive        *bool   `json:"isActive"`
}
type createStaffRequest struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	Title             *string                     `json:"title"`
	CommissionRate    *float64                    `json:"commissionRate"`
	IsActive          *bool                       `json:"isActive"`
	AvailabilitySlots []createAvailabilityRequest `json:"availabilitySlots"`
}
type createAvailabilityRequest struct {
	ID           string `json:"id"`
	Weekday      string `json:"weekday"`
	StartMinutes int16  `json:"startMinutes"`
	EndMinutes   int16  `json:"endMinutes"`
}

func (r createBusinessRequest) toDomain() (domain.CreateInput, error) {
	input := domain.CreateInput{Type: domain.BusinessType(r.Type), Name: r.Name, CategoryID: r.CategoryID, ShortDescription: r.ShortDescription, Location: domain.Location{City: r.Location.City, Address: r.Location.Address, Latitude: r.Location.Latitude, Longitude: r.Location.Longitude}, FeaturedCollectionIDs: r.FeaturedCollectionIDs}
	// Flutter names the enum values plural; the domain keeps the database-facing singular values.
	if r.Type == "stays" {
		input.Type = domain.BusinessTypeStay
	}
	if r.Type == "services" {
		input.Type = domain.BusinessTypeService
	}
	if r.LogoURL != nil {
		input.Media = append(input.Media, domain.Media{MediaType: "logo", StoragePath: *r.LogoURL})
	}
	if r.CoverPhotoURL != nil {
		input.Media = append(input.Media, domain.Media{MediaType: "cover", StoragePath: *r.CoverPhotoURL})
	}
	for position, path := range r.PhotoURLs {
		input.Media = append(input.Media, domain.Media{MediaType: "gallery", StoragePath: path, Position: int16Ptr(int16(position))})
	}
	if r.StayDetails != nil {
		stay := domain.Stay{InventoryType: r.StayDetails.InventoryType, BasePriceMinor: r.StayDetails.PricePerNight, Amenities: r.StayDetails.Amenities}
		if stay.InventoryType == "singleUnit" {
			stay.InventoryType = "single_unit"
		}
		if stay.InventoryType == "multipleUnits" {
			stay.InventoryType = "multiple_units"
		}
		for _, room := range r.StayDetails.Rooms {
			stay.UnitTypes = append(stay.UnitTypes, domain.UnitType{ClientID: room.ID, Name: room.Name, MaxGuests: room.MaxGuests, SizeSquareMeters: room.SizeSquareMeters, PricePerNightMinor: room.PricePerNight, Quantity: room.Quantity, IsActive: room.IsActive})
		}
		for _, extra := range r.StayDetails.Extras {
			unit := "one_time"
			if extra.IsPerHour {
				unit = "per_hour"
			} else if extra.IsPerNight {
				unit = "per_night"
			}
			stay.Extras = append(stay.Extras, domain.Extra{Type: extra.Type, PriceMinor: extra.Price, PricingUnit: unit})
		}
		input.Stay = &stay
	}
	if r.ServiceDetails != nil {
		service := domain.Service{TimeZone: "Europe/Sarajevo"}
		for _, offering := range r.ServiceDetails.Offerings {
			service.Offerings = append(service.Offerings, domain.Offering{ClientID: offering.ID, Name: offering.Name, Description: offering.Description, DurationMinutes: offering.DurationMinutes, PriceMinor: offering.Price, IsActive: offering.IsActive})
		}
		staff := r.ServiceDetails.Providers
		if len(staff) == 0 && r.ServiceDetails.Provider != nil {
			staff = []createStaffRequest{*r.ServiceDetails.Provider}
		}
		for _, member := range staff {
			mapped, err := member.toDomain(service.Offerings)
			if err != nil {
				return domain.CreateInput{}, err
			}
			service.Staff = append(service.Staff, mapped)
		}
		input.Service = &service
	}
	return input, nil
}

func (r createStaffRequest) toDomain(offerings []domain.Offering) (domain.Staff, error) {
	staff := domain.Staff{ClientID: r.ID, Name: r.Name, Title: r.Title, CommissionRate: r.CommissionRate, IsActive: r.IsActive}
	for _, offering := range offerings {
		staff.OfferingClientIDs = append(staff.OfferingClientIDs, offering.ClientID)
	}
	for _, slot := range r.AvailabilitySlots {
		weekday, ok := map[string]int16{"monday": 0, "tuesday": 1, "wednesday": 2, "thursday": 3, "friday": 4, "saturday": 5, "sunday": 6}[slot.Weekday]
		if !ok {
			return domain.Staff{}, fmt.Errorf("invalid availability weekday")
		}
		staff.WeeklyAvailability = append(staff.WeeklyAvailability, domain.WeeklyAvailability{ClientID: slot.ID, Weekday: weekday, StartMinutes: slot.StartMinutes, EndMinutes: slot.EndMinutes})
	}
	return staff, nil
}

func int16Ptr(value int16) *int16 { return &value }
