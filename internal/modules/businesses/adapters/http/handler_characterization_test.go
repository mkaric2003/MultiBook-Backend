package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

func TestBusinessRESTCharacterization(t *testing.T) {
	businessID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	roomID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	createdAt := time.Date(2026, time.August, 30, 10, 0, 0, 0, time.UTC)
	provider := providerUser("provider-1", "Sarajevo")

	t.Run("create business accepts Flutter shape and returns full Flutter model", func(t *testing.T) {
		repository := &businessRepositoryStub{createBusiness: business(businessID, createdAt)}
		request := authenticatedRequest(t, http.MethodPost, "/businesses", `{
			"type":"stays", "name":"Hotel Europe", "categoryId":"hotel",
			"location":{"city":"Sarajevo","address":"Ferhadija 1","latitude":43.86,"longitude":18.42},
			"logoUrl":"logos/europe.webp", "coverPhotoUrl":"covers/europe.webp", "photoUrls":["photos/1.webp"],
			"featuredCollectionIds":["featured"],
			"stayDetails":{"pricePerNight":12000,"inventoryType":"singleUnit","amenities":["wifi"],"rooms":[{"name":"Deluxe","maxGuests":2,"sizeSquareMeters":25,"pricePerNight":12000,"quantity":1}],"extras":[{"type":"breakfast","price":2000,"isPerNight":true}]}
		}`, provider)
		recorder := httptest.NewRecorder()
		NewHandler(application.NewService(repository, repository)).Create(recorder, request)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}
		body := responseObject(t, recorder)
		if body["id"] != businessID.String() || body["type"] != "stays" || body["categoryId"] != "hotel" || body["coverPhotoUrl"] != "covers/europe.webp" {
			t.Fatalf("unexpected create response: %#v", body)
		}
		stay := body["stayDetails"].(map[string]any)
		if stay["inventoryType"] != "singleUnit" || stay["rooms"].([]any)[0].(map[string]any)["id"] != roomID.String() {
			t.Fatalf("unexpected stay response: %#v", stay)
		}
		if repository.createInput.Type != domain.BusinessTypeStay || repository.createInput.Stay.UnitTypes[0].ID != uuid.Nil {
			t.Fatalf("repository mutated create input: %#v", repository.createInput)
		}
	})

	t.Run("owned summary list is wrapped in items and preserves API-shaped models", func(t *testing.T) {
		price := int64(12000)
		repository := &businessRepositoryStub{owned: []application.OwnedSummary{{BusinessReadBase: readBase(businessID, createdAt), Stay: &application.StaySummaryRead{PricePerNight: &price, InventoryType: "single_unit"}}}}
		recorder := httptest.NewRecorder()
		NewHandler(application.NewService(repository, repository)).ListOwned(recorder, authenticatedRequest(t, http.MethodGet, "/businesses", "", provider))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		items := responseObject(t, recorder)["items"].([]any)
		item := items[0].(map[string]any)
		if item["coverPhotoUrl"] != "covers/europe.webp" || item["stayDetails"].(map[string]any)["pricePerNight"] != float64(12000) {
			t.Fatalf("unexpected owned summary: %#v", item)
		}
	})

	t.Run("owner detail returns the complete API-shaped model directly", func(t *testing.T) {
		repository := &businessRepositoryStub{ownedDetail: application.OwnedDetail{BusinessReadBase: serviceReadBase(businessID, createdAt), Service: &application.ServiceDetailRead{Offerings: []application.ServiceOfferingRead{{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333"), Name: "Massage", DurationMinutes: 30}}}}}
		recorder := httptest.NewRecorder()
		NewHandler(application.NewService(repository, repository)).GetOwned(recorder, requestWithBusinessID(t, http.MethodGet, businessID, "", provider))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		body := responseObject(t, recorder)
		if body["type"] != "services" || body["serviceDetails"].(map[string]any)["offerings"].([]any)[0].(map[string]any)["durationMinutes"] != float64(30) {
			t.Fatalf("unexpected owner detail: %#v", body)
		}
	})

	t.Run("update accepts snake_case patch fields and returns the persisted business", func(t *testing.T) {
		name := "Renamed Hotel"
		status := domain.BusinessStatusInactive
		repository := &businessRepositoryStub{updatedBusiness: domain.Business{ID: businessID, OwnerID: "provider-1", Type: domain.BusinessTypeStay, Status: status, Name: name, CategoryID: "hotel", Currency: "BAM", CreatedAt: createdAt, UpdatedAt: createdAt}}
		recorder := httptest.NewRecorder()
		NewHandler(application.NewService(repository, repository)).Update(recorder, requestWithBusinessID(t, http.MethodPatch, businessID, `{"name":"  Renamed Hotel  ","status":"inactive"}`, provider))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		body := responseObject(t, recorder)
		if body["name"] != name || body["status"] != "inactive" || body["category_id"] != "hotel" {
			t.Fatalf("unexpected update response: %#v", body)
		}
		if repository.updateInput.Name == nil || *repository.updateInput.Name != name || repository.updateID != businessID {
			t.Fatalf("unexpected update input: %#v", repository)
		}
	})
}

type businessRepositoryStub struct {
	createBusiness  domain.Business
	createInput     domain.CreateInput
	owned           []application.OwnedSummary
	ownedDetail     application.OwnedDetail
	updatedBusiness domain.Business
	updateID        uuid.UUID
	updateInput     domain.UpdateInput
}

func (r *businessRepositoryStub) Create(_ context.Context, _ application.Actor, input domain.CreateInput) (application.PersistedBusiness, error) {
	r.createInput = input
	return application.PersistedBusiness{Business: r.createBusiness, AggregateIDs: application.AggregateIDMapping{StayUnitTypeIDs: []uuid.UUID{uuid.MustParse("22222222-2222-2222-2222-222222222222")}}}, nil
}
func (*businessRepositoryStub) ReplaceOwned(context.Context, application.Actor, uuid.UUID, domain.CreateInput) (application.PersistedBusiness, error) {
	return application.PersistedBusiness{}, nil
}
func (r *businessRepositoryStub) ListOwned(context.Context, string) ([]application.OwnedSummary, error) {
	return r.owned, nil
}
func (r *businessRepositoryStub) GetOwnedDetail(context.Context, string, uuid.UUID) (application.OwnedDetail, error) {
	return r.ownedDetail, nil
}
func (r *businessRepositoryStub) UpdateOwned(_ context.Context, _ application.Actor, id uuid.UUID, input domain.UpdateInput) (domain.Business, error) {
	r.updateID, r.updateInput = id, input
	return r.updatedBusiness, nil
}
func (*businessRepositoryStub) ArchiveOwned(context.Context, application.Actor, uuid.UUID) error {
	return nil
}
func (*businessRepositoryStub) SetSelectedBusiness(context.Context, application.Actor, uuid.UUID) error {
	return nil
}
func business(id uuid.UUID, timestamp time.Time) domain.Business {
	return domain.Business{ID: id, OwnerID: "provider-1", Type: domain.BusinessTypeStay, Status: domain.BusinessStatusActive, Name: "Hotel Europe", CategoryID: "hotel", Currency: "BAM", CreatedAt: timestamp, UpdatedAt: timestamp}
}

func readBase(id uuid.UUID, timestamp time.Time) application.BusinessReadBase {
	cover := "covers/europe.webp"
	return application.BusinessReadBase{ID: id, OwnerID: "provider-1", Type: domain.BusinessTypeStay, Status: domain.BusinessStatusActive, Name: "Hotel Europe", CategoryID: "hotel", Location: application.LocationRead{City: "Sarajevo", Address: "Ferhadija 1", Latitude: 43.86, Longitude: 18.42}, Currency: "BAM", CoverPath: &cover, PhotoPaths: []string{}, FeaturedCollectionIDs: []string{}, CreatedAt: timestamp, UpdatedAt: timestamp}
}

func serviceReadBase(id uuid.UUID, timestamp time.Time) application.BusinessReadBase {
	base := readBase(id, timestamp)
	base.Type, base.Name = domain.BusinessTypeService, "Spa"
	return base
}

func providerUser(id, city string) usersdomain.User {
	role := usersdomain.UserRoleProvider
	return usersdomain.User{ID: id, Role: &role, City: &city, BusinessCurrency: "BAM"}
}

func authenticatedRequest(t *testing.T, method, target, body string, user usersdomain.User) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return usershttp.WithCurrentUser(request, user)
}

func requestWithBusinessID(t *testing.T, method string, id uuid.UUID, body string, user usersdomain.User) *http.Request {
	t.Helper()
	request := authenticatedRequest(t, method, "/businesses/"+id.String(), body, user)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("businessID", id.String())
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func responseObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response JSON: %v; body=%s", err, recorder.Body.String())
	}
	return body
}
