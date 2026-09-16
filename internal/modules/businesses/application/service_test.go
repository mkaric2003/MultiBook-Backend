package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

type repositoryStub struct {
	createActor  applicationActorCapture
	createInput  domain.CreateInput
	persisted    PersistedBusiness
	owned        OwnedDetail
	ownedErr     error
	queryCalls   int
	replaceCalls int
	updateCalls  int
}

type applicationActorCapture struct {
	id       string
	currency string
}

func (r *repositoryStub) Create(_ context.Context, actor Actor, input domain.CreateInput) (PersistedBusiness, error) {
	r.createActor = applicationActorCapture{id: actor.ID, currency: actor.Currency}
	r.createInput = input
	return r.persisted, nil
}
func (r *repositoryStub) ReplaceOwned(context.Context, Actor, uuid.UUID, domain.CreateInput) (PersistedBusiness, error) {
	r.replaceCalls++
	return r.persisted, nil
}

func (*repositoryStub) ListOwned(context.Context, string) ([]OwnedSummary, error) {
	return nil, nil
}

func (r *repositoryStub) GetOwnedDetail(context.Context, string, uuid.UUID) (OwnedDetail, error) {
	r.queryCalls++
	return r.owned, r.ownedErr
}

func (r *repositoryStub) UpdateOwned(context.Context, Actor, uuid.UUID, domain.UpdateInput) (domain.Business, error) {
	r.updateCalls++
	return domain.Business{}, nil
}

func (*repositoryStub) ArchiveOwned(context.Context, Actor, uuid.UUID) error { return nil }

func (*repositoryStub) SetSelectedBusiness(context.Context, Actor, uuid.UUID) error { return nil }

func TestCreateRejectsCustomer(t *testing.T) {
	stub := &repositoryStub{}
	service := NewService(stub, stub)
	_, err := service.Create(context.Background(), Actor{ID: "customer", Role: "customer", Currency: "BAM"}, domain.CreateInput{
		Type:       domain.BusinessTypeStay,
		Name:       "Stay",
		CategoryID: "hotel",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want forbidden", err)
	}
}

func TestCreateRejectsAdminWithoutAnExplicitAdminPolicy(t *testing.T) {
	t.Parallel()
	stub := &repositoryStub{}
	_, err := NewService(stub, stub).Create(context.Background(), Actor{ID: "admin", Role: "admin", Currency: "BAM"}, validStayInput())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want forbidden", err)
	}
}

func TestCreateNormalizesProviderInput(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, repository)
	name, categoryID := "  Hotel Europe  ", " hotel "
	city, address, countryCode := "  Sarajevo  ", "  Address  ", " BA "
	input := domain.CreateInput{
		Type:       domain.BusinessTypeStay,
		Name:       name,
		CategoryID: categoryID,
		Location:   domain.Location{City: city, Address: address, CountryCode: &countryCode, Latitude: 43.85, Longitude: 18.41},
		Stay:       &domain.Stay{InventoryType: "single_unit", UnitTypes: []domain.UnitType{{Name: "Room"}}},
	}
	repository.persisted.AggregateIDs.StayUnitTypeIDs = []uuid.UUID{uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	created, err := service.Create(context.Background(), Actor{ID: "provider", Role: "provider", Currency: "BAM"}, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.createInput.Name != "Hotel Europe" || repository.createInput.CategoryID != "hotel" {
		t.Fatalf("Create() input = %#v, want trimmed values", repository.createInput)
	}
	if repository.createInput.Location.City != "Sarajevo" || repository.createInput.Location.Address != "Address" || repository.createInput.Location.CountryCode == nil || *repository.createInput.Location.CountryCode != "BA" {
		t.Fatalf("Create() location = %#v, want trimmed values", repository.createInput.Location)
	}
	if input.Name != name || input.CategoryID != categoryID || input.Stay.UnitTypes[0].ID != uuid.Nil {
		t.Fatalf("Create() mutated caller input: %#v", input)
	}
	if created.Input.Stay.UnitTypes[0].ID != repository.persisted.AggregateIDs.StayUnitTypeIDs[0] {
		t.Fatalf("Create() result did not apply generated ID: %#v", created)
	}
}

func TestUpdateRejectsArchivedStatus(t *testing.T) {
	stub := &repositoryStub{}
	service := NewService(stub, stub)
	status := domain.BusinessStatusArchived
	_, err := service.Update(context.Background(), Actor{ID: "provider", Role: "provider", Currency: "BAM"}, uuid.New(), domain.UpdateInput{Status: &status})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want validation error", err)
	}
}

func TestWithAggregateIDsKeepsExistingAggregateIdentities(t *testing.T) {
	t.Parallel()
	unitID, offeringID, staffID, availabilityID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	input := domain.CreateInput{
		Stay:    &domain.Stay{UnitTypes: []domain.UnitType{{ClientID: unitID.String()}}},
		Service: &domain.Service{Offerings: []domain.Offering{{ClientID: offeringID.String()}}, Staff: []domain.Staff{{ClientID: staffID.String(), WeeklyAvailability: []domain.WeeklyAvailability{{ClientID: availabilityID.String()}}}}},
	}
	result := withAggregateIDs(input, AggregateIDMapping{StayUnitTypeIDs: []uuid.UUID{unitID}, ServiceOfferingIDs: []uuid.UUID{offeringID}, ServiceStaffIDs: []uuid.UUID{staffID}, ServiceStaffAvailabilityIDs: [][]uuid.UUID{{availabilityID}}})
	if result.Stay.UnitTypes[0].ID != unitID || result.Service.Offerings[0].ID != offeringID || result.Service.Staff[0].ID != staffID || result.Service.Staff[0].WeeklyAvailability[0].ID != availabilityID {
		t.Fatalf("aggregate identities changed: %#v", result)
	}
}

func TestReplaceRejectsTypeChangeBeforeNestedCommand(t *testing.T) {
	t.Parallel()
	stub := &repositoryStub{owned: OwnedDetail{BusinessReadBase: BusinessReadBase{Type: domain.BusinessTypeStay, Status: domain.BusinessStatusActive}}}
	_, err := NewService(stub, stub).Replace(context.Background(), providerActor(), uuid.New(), validServiceInput())
	if !errors.Is(err, ErrTypeImmutable) {
		t.Fatalf("Replace() error = %v, want immutable type", err)
	}
	if stub.queryCalls != 1 || stub.replaceCalls != 0 {
		t.Fatalf("Replace() must check ownership/lifecycle before nested command: %#v", stub)
	}
}

func TestReplaceRejectsArchivedBusinessBeforeNestedCommand(t *testing.T) {
	t.Parallel()
	stub := &repositoryStub{owned: OwnedDetail{BusinessReadBase: BusinessReadBase{Type: domain.BusinessTypeStay, Status: domain.BusinessStatusArchived}}}
	_, err := NewService(stub, stub).Replace(context.Background(), providerActor(), uuid.New(), validStayInput())
	if !errors.Is(err, ErrBusinessArchived) {
		t.Fatalf("Replace() error = %v, want archived lifecycle error", err)
	}
	if stub.queryCalls != 1 || stub.replaceCalls != 0 {
		t.Fatalf("Replace() must not mutate archived aggregate: %#v", stub)
	}
}

func TestUpdateChecksOwnershipAndLifecycleBeforeCommand(t *testing.T) {
	t.Parallel()
	name := "Updated"
	stub := &repositoryStub{owned: OwnedDetail{BusinessReadBase: BusinessReadBase{Type: domain.BusinessTypeStay, Status: domain.BusinessStatusArchived}}}
	_, err := NewService(stub, stub).Update(context.Background(), providerActor(), uuid.New(), domain.UpdateInput{Name: &name})
	if !errors.Is(err, ErrBusinessArchived) {
		t.Fatalf("Update() error = %v, want archived lifecycle error", err)
	}
	if stub.queryCalls != 1 || stub.updateCalls != 0 {
		t.Fatalf("Update() must check ownership/lifecycle before command: %#v", stub)
	}
}

func TestReplaceChecksOwnershipBeforeNestedCommand(t *testing.T) {
	t.Parallel()
	stub := &repositoryStub{ownedErr: ErrBusinessNotFound}
	_, err := NewService(stub, stub).Replace(context.Background(), providerActor(), uuid.New(), validStayInput())
	if !errors.Is(err, ErrBusinessNotFound) {
		t.Fatalf("Replace() error = %v, want ownership not found", err)
	}
	if stub.queryCalls != 1 || stub.replaceCalls != 0 {
		t.Fatalf("Replace() must check ownership before nested command: %#v", stub)
	}
}

func providerActor() Actor { return Actor{ID: "provider", Role: "provider", Currency: "BAM"} }

func validStayInput() domain.CreateInput {
	return domain.CreateInput{Type: domain.BusinessTypeStay, Name: "Stay", CategoryID: "hotel", Location: domain.Location{City: "Sarajevo", Address: "Main 1", Latitude: 43.85, Longitude: 18.41}, Stay: &domain.Stay{InventoryType: "single_unit"}}
}

func validServiceInput() domain.CreateInput {
	return domain.CreateInput{Type: domain.BusinessTypeService, Name: "Service", CategoryID: "salon", Location: domain.Location{City: "Sarajevo", Address: "Main 1", Latitude: 43.85, Longitude: 18.41}, Service: &domain.Service{TimeZone: "Europe/Sarajevo", Offerings: []domain.Offering{{Name: "Cut"}}, Staff: []domain.Staff{{Name: "Amina"}}}}
}
