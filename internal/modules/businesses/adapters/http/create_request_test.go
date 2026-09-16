package http

import (
	"testing"

	"github.com/google/uuid"
)

func TestCreateRequestCarriesPersistedAggregateIDsForReplace(t *testing.T) {
	t.Parallel()
	roomID, offeringID, staffID, slotID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	request := createBusinessRequest{
		Type:       "services",
		Name:       "Service",
		CategoryID: "salon",
		Location:   createLocationRequest{City: "Sarajevo", Address: "Main 1", Latitude: 43.85, Longitude: 18.41},
		ServiceDetails: &createServiceRequest{
			Offerings: []createOfferingRequest{{ID: offeringID, Name: "Cut", DurationMinutes: 30, Price: 1000}},
			Providers: []createStaffRequest{{ID: staffID, Name: "Amina", AvailabilitySlots: []createAvailabilityRequest{{ID: slotID, Weekday: "monday", StartMinutes: 540, EndMinutes: 600}}}},
		},
	}
	input, err := request.toDomain()
	if err != nil {
		t.Fatal(err)
	}
	if input.Service.Offerings[0].ClientID != offeringID || input.Service.Staff[0].ClientID != staffID || input.Service.Staff[0].WeeklyAvailability[0].ClientID != slotID {
		t.Fatalf("persisted service IDs were not preserved: %#v", input.Service)
	}

	request.Type = "stays"
	request.ServiceDetails = nil
	request.StayDetails = &createStayRequest{InventoryType: "multipleUnits", Rooms: []createRoomRequest{{ID: roomID, Name: "Room", MaxGuests: 2, Quantity: 1}}}
	input, err = request.toDomain()
	if err != nil {
		t.Fatal(err)
	}
	if input.Stay.UnitTypes[0].ClientID != roomID {
		t.Fatalf("persisted unit type ID was not preserved: %#v", input.Stay.UnitTypes[0])
	}
}
