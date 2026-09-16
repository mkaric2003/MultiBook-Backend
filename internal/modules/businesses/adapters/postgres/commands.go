package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.BusinessRepository = (*CommandRepository)(nil)

const cols = "id,owner_id,type,status,name,category_id,currency,short_description,average_rating,review_count,is_promotion_active,created_at,updated_at"

func (r *CommandRepository) Create(ctx context.Context, a application.Actor, i domain.CreateInput) (application.PersistedBusiness, error) {
	result := application.PersistedBusiness{}
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		business, err := scan(tx.QueryRow(ctx, "INSERT INTO businesses (owner_id,type,name,name_normalized,category_id,currency,short_description) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING "+cols, a.ID, i.Type, i.Name, domain.NormalizeSearchText(i.Name), i.CategoryID, a.Currency, i.ShortDescription))
		result.Business = business
		if err != nil {
			return err
		}
		ids, err := createAggregate(ctx, tx, business.ID, i)
		if err != nil {
			return err
		}
		result.AggregateIDs = ids
		return r.writeAudit(ctx, tx, business.ID, a.ID, "business_created")
	})
	return result, err
}

func (r *CommandRepository) ReplaceOwned(ctx context.Context, a application.Actor, id uuid.UUID, i domain.CreateInput) (application.PersistedBusiness, error) {
	result := application.PersistedBusiness{}
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		business, err := scan(tx.QueryRow(ctx, "UPDATE businesses SET type=$3,name=$4,name_normalized=$5,category_id=$6,short_description=$7 WHERE id=$1 AND owner_id=$2 AND deleted_at IS NULL RETURNING "+cols, id, a.ID, i.Type, i.Name, domain.NormalizeSearchText(i.Name), i.CategoryID, i.ShortDescription))
		result.Business = business
		if err != nil {
			return err
		}
		ids, err := replaceAggregate(ctx, tx, id, i)
		if err != nil {
			return err
		}
		result.AggregateIDs = ids
		return r.writeAudit(ctx, tx, id, a.ID, "business_replaced")
	})
	return result, err
}

func createAggregate(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, input domain.CreateInput) (application.AggregateIDMapping, error) {
	ids := application.AggregateIDMapping{}
	location := input.Location
	if _, err := tx.Exec(ctx, `INSERT INTO business_locations(business_id,city,city_normalized,address,country_code,coordinates)VALUES($1,$2,$3,$4,$5,ST_SetSRID(ST_MakePoint($6,$7),4326)::geography)`, businessID, location.City, domain.NormalizeSearchText(location.City), location.Address, location.CountryCode, location.Longitude, location.Latitude); err != nil {
		return ids, err
	}
	for _, media := range input.Media {
		if _, err := tx.Exec(ctx, `INSERT INTO business_media(business_id,media_type,storage_path,position)VALUES($1,$2::business_media_type,$3,$4)`, businessID, media.MediaType, media.StoragePath, media.Position); err != nil {
			return ids, err
		}
	}
	for _, collectionID := range input.FeaturedCollectionIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO business_featured_collections(business_id,collection_id)VALUES($1,$2)`, businessID, collectionID); err != nil {
			return ids, err
		}
	}
	if input.Type == domain.BusinessTypeStay {
		unitIDs, err := createStay(ctx, tx, businessID, input.Stay)
		ids.StayUnitTypeIDs = unitIDs
		return ids, err
	}
	return createService(ctx, tx, businessID, input.Service)
}

// replaceAggregate updates the editable aggregate in place. Server UUIDs from
// the Flutter payload identify existing unit types, offerings, staff and
// availability slots; local draft IDs identify new rows. Rows omitted from a
// replace are retired rather than deleted when they can be historically
// referenced by a booking or appointment.
func replaceAggregate(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, input domain.CreateInput) (application.AggregateIDMapping, error) {
	if err := upsertLocation(ctx, tx, businessID, input.Location); err != nil {
		return application.AggregateIDMapping{}, err
	}
	if err := replaceMedia(ctx, tx, businessID, input.Media); err != nil {
		return application.AggregateIDMapping{}, err
	}
	if err := replaceFeaturedCollections(ctx, tx, businessID, input.FeaturedCollectionIDs); err != nil {
		return application.AggregateIDMapping{}, err
	}
	if input.Type == domain.BusinessTypeStay {
		if err := retireServiceAggregate(ctx, tx, businessID); err != nil {
			return application.AggregateIDMapping{}, err
		}
		unitIDs, err := reconcileStay(ctx, tx, businessID, input.Stay)
		return application.AggregateIDMapping{StayUnitTypeIDs: unitIDs}, err
	}
	if err := retireStayAggregate(ctx, tx, businessID); err != nil {
		return application.AggregateIDMapping{}, err
	}
	return reconcileService(ctx, tx, businessID, input.Service)
}

func upsertLocation(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, location domain.Location) error {
	_, err := tx.Exec(ctx, `INSERT INTO business_locations(business_id,city,city_normalized,address,country_code,coordinates,is_primary)VALUES($1,$2,$3,$4,$5,ST_SetSRID(ST_MakePoint($6,$7),4326)::geography,TRUE) ON CONFLICT (business_id) WHERE is_primary DO UPDATE SET city=EXCLUDED.city,city_normalized=EXCLUDED.city_normalized,address=EXCLUDED.address,country_code=EXCLUDED.country_code,coordinates=EXCLUDED.coordinates`, businessID, location.City, domain.NormalizeSearchText(location.City), location.Address, location.CountryCode, location.Longitude, location.Latitude)
	return err
}

func replaceMedia(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, media []domain.Media) error {
	// Media has no booking or appointment reference. It has no client-visible
	// stable ID, so its type/position is the aggregate key.
	if _, err := tx.Exec(ctx, "DELETE FROM business_media WHERE business_id=$1", businessID); err != nil {
		return err
	}
	for _, item := range media {
		if _, err := tx.Exec(ctx, `INSERT INTO business_media(business_id,media_type,storage_path,position)VALUES($1,$2::business_media_type,$3,$4)`, businessID, item.MediaType, item.StoragePath, item.Position); err != nil {
			return err
		}
	}
	return nil
}

func replaceFeaturedCollections(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, collectionIDs []string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM business_featured_collections WHERE business_id=$1", businessID); err != nil {
		return err
	}
	for _, collectionID := range collectionIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO business_featured_collections(business_id,collection_id)VALUES($1,$2)`, businessID, collectionID); err != nil {
			return err
		}
	}
	return nil
}

func retireStayAggregate(ctx context.Context, tx pgx.Tx, businessID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `UPDATE stay_units SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND deleted_at IS NULL`, businessID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE stay_unit_types SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND deleted_at IS NULL`, businessID)
	return err
}

func retireServiceAggregate(ctx context.Context, tx pgx.Tx, businessID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `UPDATE service_staff SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND deleted_at IS NULL`, businessID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE service_offerings SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND deleted_at IS NULL`, businessID)
	return err
}
func createStay(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, stay *domain.Stay) ([]uuid.UUID, error) {
	if stay == nil {
		return nil, application.ErrValidation
	}
	unitIDs := make([]uuid.UUID, 0, len(stay.UnitTypes))
	if _, err := tx.Exec(ctx, `INSERT INTO stay_details(business_id,inventory_type,base_price_minor)VALUES($1,$2::stay_inventory_type,$3)`, businessID, stay.InventoryType, stay.BasePriceMinor); err != nil {
		return unitIDs, err
	}
	for _, amenity := range stay.Amenities {
		if _, err := tx.Exec(ctx, `INSERT INTO stay_amenities(business_id,amenity_code)VALUES($1,$2)`, businessID, amenity); err != nil {
			return unitIDs, err
		}
	}
	for _, extra := range stay.Extras {
		if _, err := tx.Exec(ctx, `INSERT INTO stay_extras(business_id,extra_type,price_minor,pricing_unit)VALUES($1,$2,$3,$4)`, businessID, extra.Type, extra.PriceMinor, extra.PricingUnit); err != nil {
			return unitIDs, err
		}
	}
	for _, unit := range stay.UnitTypes {
		active := true
		if unit.IsActive != nil {
			active = *unit.IsActive
		}
		var unitID uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO stay_unit_types(business_id,name,max_guests,size_square_meters,price_per_night_minor,quantity,is_active)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING id`, businessID, unit.Name, unit.MaxGuests, unit.SizeSquareMeters, unit.PricePerNightMinor, unit.Quantity, active).Scan(&unitID)
		if err != nil {
			return unitIDs, err
		}
		unitIDs = append(unitIDs, unitID)
		for sequence := int16(1); sequence <= unit.Quantity; sequence++ {
			if _, err = tx.Exec(ctx, `INSERT INTO stay_units(business_id,stay_unit_type_id,sequence_number,is_active)VALUES($1,$2,$3,$4)`, businessID, unitID, sequence, active); err != nil {
				return unitIDs, err
			}
		}
	}
	return unitIDs, nil
}
func createService(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, service *domain.Service) (application.AggregateIDMapping, error) {
	if service == nil {
		return application.AggregateIDMapping{}, application.ErrValidation
	}
	ids := application.AggregateIDMapping{ServiceOfferingIDs: make([]uuid.UUID, 0, len(service.Offerings)), ServiceStaffIDs: make([]uuid.UUID, 0, len(service.Staff)), ServiceStaffAvailabilityIDs: make([][]uuid.UUID, 0, len(service.Staff))}
	if _, err := tx.Exec(ctx, `INSERT INTO service_details(business_id,time_zone)VALUES($1,$2)`, businessID, service.TimeZone); err != nil {
		return ids, err
	}
	offeringIDs := map[string]uuid.UUID{}
	for _, offering := range service.Offerings {
		active := true
		if offering.IsActive != nil {
			active = *offering.IsActive
		}
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO service_offerings(business_id,name,description,duration_minutes,price_minor,is_active)VALUES($1,$2,$3,$4,$5,$6)RETURNING id`, businessID, offering.Name, offering.Description, offering.DurationMinutes, offering.PriceMinor, active).Scan(&id)
		if err != nil {
			return ids, err
		}
		ids.ServiceOfferingIDs = append(ids.ServiceOfferingIDs, id)
		offeringIDs[offering.ClientID] = id
	}
	for _, staff := range service.Staff {
		active := true
		if staff.IsActive != nil {
			active = *staff.IsActive
		}
		commission := float64(100)
		if staff.CommissionRate != nil {
			commission = *staff.CommissionRate
		}
		var staffID uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO service_staff(business_id,name,title,commission_rate,is_active)VALUES($1,$2,$3,$4,$5)RETURNING id`, businessID, staff.Name, staff.Title, commission, active).Scan(&staffID)
		if err != nil {
			return ids, err
		}
		ids.ServiceStaffIDs = append(ids.ServiceStaffIDs, staffID)
		availabilityIDs := make([]uuid.UUID, 0, len(staff.WeeklyAvailability))
		for _, clientID := range staff.OfferingClientIDs {
			id, ok := offeringIDs[clientID]
			if !ok {
				return ids, application.ErrValidation
			}
			if _, err = tx.Exec(ctx, `INSERT INTO service_staff_offerings(staff_id,offering_id)VALUES($1,$2)`, staffID, id); err != nil {
				return ids, err
			}
		}
		for _, availability := range staff.WeeklyAvailability {
			var availabilityID uuid.UUID
			if err = tx.QueryRow(ctx, `INSERT INTO service_staff_weekly_availability(staff_id,weekday,start_minutes,end_minutes)VALUES($1,$2,$3,$4) RETURNING id`, staffID, availability.Weekday, availability.StartMinutes, availability.EndMinutes).Scan(&availabilityID); err != nil {
				return ids, err
			}
			availabilityIDs = append(availabilityIDs, availabilityID)
		}
		ids.ServiceStaffAvailabilityIDs = append(ids.ServiceStaffAvailabilityIDs, availabilityIDs)
	}
	return ids, nil
}

func reconcileStay(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, stay *domain.Stay) ([]uuid.UUID, error) {
	if stay == nil {
		return nil, application.ErrValidation
	}
	if _, err := tx.Exec(ctx, `INSERT INTO stay_details(business_id,inventory_type,base_price_minor)VALUES($1,$2::stay_inventory_type,$3) ON CONFLICT (business_id) DO UPDATE SET inventory_type=EXCLUDED.inventory_type,base_price_minor=EXCLUDED.base_price_minor`, businessID, stay.InventoryType, stay.BasePriceMinor); err != nil {
		return nil, err
	}
	// Amenities and extras are value objects with no booking references.
	if _, err := tx.Exec(ctx, "DELETE FROM stay_amenities WHERE business_id=$1", businessID); err != nil {
		return nil, err
	}
	for _, amenity := range stay.Amenities {
		if _, err := tx.Exec(ctx, `INSERT INTO stay_amenities(business_id,amenity_code)VALUES($1,$2)`, businessID, amenity); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM stay_extras WHERE business_id=$1", businessID); err != nil {
		return nil, err
	}
	for _, extra := range stay.Extras {
		if _, err := tx.Exec(ctx, `INSERT INTO stay_extras(business_id,extra_type,price_minor,pricing_unit)VALUES($1,$2,$3,$4)`, businessID, extra.Type, extra.PriceMinor, extra.PricingUnit); err != nil {
			return nil, err
		}
	}
	ids := make([]uuid.UUID, 0, len(stay.UnitTypes))
	kept := make([]uuid.UUID, 0, len(stay.UnitTypes))
	seen := map[uuid.UUID]struct{}{}
	for _, unit := range stay.UnitTypes {
		unitID, existing := persistedID(unit.ClientID)
		if existing {
			if _, duplicate := seen[unitID]; duplicate {
				return nil, application.ErrValidation
			}
			seen[unitID] = struct{}{}
		}
		active := boolValue(unit.IsActive, true)
		if existing {
			command, err := tx.Exec(ctx, `UPDATE stay_unit_types SET name=$3,max_guests=$4,size_square_meters=$5,price_per_night_minor=$6,quantity=$7,is_active=$8,deleted_at=NULL WHERE id=$1 AND business_id=$2`, unitID, businessID, unit.Name, unit.MaxGuests, unit.SizeSquareMeters, unit.PricePerNightMinor, unit.Quantity, active)
			if err != nil {
				return nil, err
			}
			if command.RowsAffected() == 0 {
				return nil, application.ErrValidation
			}
		} else if err := tx.QueryRow(ctx, `INSERT INTO stay_unit_types(business_id,name,max_guests,size_square_meters,price_per_night_minor,quantity,is_active)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING id`, businessID, unit.Name, unit.MaxGuests, unit.SizeSquareMeters, unit.PricePerNightMinor, unit.Quantity, active).Scan(&unitID); err != nil {
			return nil, err
		}
		if err := synchronizeUnitTypeUnits(ctx, tx, businessID, unitID, unit.Quantity, active); err != nil {
			return nil, err
		}
		ids = append(ids, unitID)
		kept = append(kept, unitID)
	}
	if _, err := tx.Exec(ctx, `UPDATE stay_units SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND stay_unit_type_id <> ALL($2::uuid[]) AND deleted_at IS NULL`, businessID, kept); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE stay_unit_types SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND id <> ALL($2::uuid[]) AND deleted_at IS NULL`, businessID, kept); err != nil {
		return nil, err
	}
	return ids, nil
}

func synchronizeUnitTypeUnits(ctx context.Context, tx pgx.Tx, businessID, unitTypeID uuid.UUID, quantity int16, active bool) error {
	for sequence := int16(1); sequence <= quantity; sequence++ {
		command, err := tx.Exec(ctx, `UPDATE stay_units SET is_active=$4,deleted_at=NULL WHERE business_id=$1 AND stay_unit_type_id=$2 AND sequence_number=$3`, businessID, unitTypeID, sequence, active)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			if _, err = tx.Exec(ctx, `INSERT INTO stay_units(business_id,stay_unit_type_id,sequence_number,is_active)VALUES($1,$2,$3,$4)`, businessID, unitTypeID, sequence, active); err != nil {
				return err
			}
		}
	}
	_, err := tx.Exec(ctx, `UPDATE stay_units SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND stay_unit_type_id=$2 AND sequence_number>$3 AND deleted_at IS NULL`, businessID, unitTypeID, quantity)
	return err
}

func reconcileService(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, service *domain.Service) (application.AggregateIDMapping, error) {
	if service == nil {
		return application.AggregateIDMapping{}, application.ErrValidation
	}
	if _, err := tx.Exec(ctx, `INSERT INTO service_details(business_id,time_zone)VALUES($1,$2) ON CONFLICT (business_id) DO UPDATE SET time_zone=EXCLUDED.time_zone`, businessID, service.TimeZone); err != nil {
		return application.AggregateIDMapping{}, err
	}
	ids := application.AggregateIDMapping{ServiceOfferingIDs: make([]uuid.UUID, 0, len(service.Offerings)), ServiceStaffIDs: make([]uuid.UUID, 0, len(service.Staff)), ServiceStaffAvailabilityIDs: make([][]uuid.UUID, 0, len(service.Staff))}
	offeringIDs := make(map[string]uuid.UUID, len(service.Offerings))
	keptOfferings := make([]uuid.UUID, 0, len(service.Offerings))
	seenOfferings := map[uuid.UUID]struct{}{}
	for _, offering := range service.Offerings {
		offeringID, existing := persistedID(offering.ClientID)
		if existing {
			if _, duplicate := seenOfferings[offeringID]; duplicate {
				return ids, application.ErrValidation
			}
			seenOfferings[offeringID] = struct{}{}
		}
		active := boolValue(offering.IsActive, true)
		if existing {
			command, err := tx.Exec(ctx, `UPDATE service_offerings SET name=$3,description=$4,duration_minutes=$5,price_minor=$6,is_active=$7,deleted_at=NULL WHERE id=$1 AND business_id=$2`, offeringID, businessID, offering.Name, offering.Description, offering.DurationMinutes, offering.PriceMinor, active)
			if err != nil {
				return ids, err
			}
			if command.RowsAffected() == 0 {
				return ids, application.ErrValidation
			}
		} else if err := tx.QueryRow(ctx, `INSERT INTO service_offerings(business_id,name,description,duration_minutes,price_minor,is_active)VALUES($1,$2,$3,$4,$5,$6)RETURNING id`, businessID, offering.Name, offering.Description, offering.DurationMinutes, offering.PriceMinor, active).Scan(&offeringID); err != nil {
			return ids, err
		}
		ids.ServiceOfferingIDs = append(ids.ServiceOfferingIDs, offeringID)
		keptOfferings = append(keptOfferings, offeringID)
		offeringIDs[offering.ClientID] = offeringID
	}
	if _, err := tx.Exec(ctx, `UPDATE service_offerings SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND id <> ALL($2::uuid[]) AND deleted_at IS NULL`, businessID, keptOfferings); err != nil {
		return ids, err
	}
	keptStaff := make([]uuid.UUID, 0, len(service.Staff))
	seenStaff := map[uuid.UUID]struct{}{}
	for _, staff := range service.Staff {
		staffID, existing := persistedID(staff.ClientID)
		if existing {
			if _, duplicate := seenStaff[staffID]; duplicate {
				return ids, application.ErrValidation
			}
			seenStaff[staffID] = struct{}{}
		}
		active, commission := boolValue(staff.IsActive, true), floatValue(staff.CommissionRate, 100)
		if existing {
			command, err := tx.Exec(ctx, `UPDATE service_staff SET name=$3,title=$4,commission_rate=$5,is_active=$6,deleted_at=NULL WHERE id=$1 AND business_id=$2`, staffID, businessID, staff.Name, staff.Title, commission, active)
			if err != nil {
				return ids, err
			}
			if command.RowsAffected() == 0 {
				return ids, application.ErrValidation
			}
		} else if err := tx.QueryRow(ctx, `INSERT INTO service_staff(business_id,name,title,commission_rate,is_active)VALUES($1,$2,$3,$4,$5)RETURNING id`, businessID, staff.Name, staff.Title, commission, active).Scan(&staffID); err != nil {
			return ids, err
		}
		if err := replaceStaffOfferings(ctx, tx, staffID, staff.OfferingClientIDs, offeringIDs); err != nil {
			return ids, err
		}
		availabilityIDs, err := reconcileWeeklyAvailability(ctx, tx, staffID, staff.WeeklyAvailability)
		if err != nil {
			return ids, err
		}
		ids.ServiceStaffIDs = append(ids.ServiceStaffIDs, staffID)
		ids.ServiceStaffAvailabilityIDs = append(ids.ServiceStaffAvailabilityIDs, availabilityIDs)
		keptStaff = append(keptStaff, staffID)
	}
	if _, err := tx.Exec(ctx, `UPDATE service_staff SET is_active=FALSE,deleted_at=COALESCE(deleted_at,now()) WHERE business_id=$1 AND id <> ALL($2::uuid[]) AND deleted_at IS NULL`, businessID, keptStaff); err != nil {
		return ids, err
	}
	return ids, nil
}

func replaceStaffOfferings(ctx context.Context, tx pgx.Tx, staffID uuid.UUID, clientIDs []string, offeringIDs map[string]uuid.UUID) error {
	if _, err := tx.Exec(ctx, "DELETE FROM service_staff_offerings WHERE staff_id=$1", staffID); err != nil {
		return err
	}
	for _, clientID := range clientIDs {
		offeringID, ok := offeringIDs[clientID]
		if !ok {
			return application.ErrValidation
		}
		if _, err := tx.Exec(ctx, `INSERT INTO service_staff_offerings(staff_id,offering_id)VALUES($1,$2)`, staffID, offeringID); err != nil {
			return err
		}
	}
	return nil
}

func reconcileWeeklyAvailability(ctx context.Context, tx pgx.Tx, staffID uuid.UUID, availability []domain.WeeklyAvailability) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(availability))
	kept := make([]uuid.UUID, 0, len(availability))
	seen := map[uuid.UUID]struct{}{}
	for _, slot := range availability {
		slotID, existing := persistedID(slot.ClientID)
		if existing {
			if _, duplicate := seen[slotID]; duplicate {
				return nil, application.ErrValidation
			}
			seen[slotID] = struct{}{}
			command, err := tx.Exec(ctx, `UPDATE service_staff_weekly_availability SET weekday=$3,start_minutes=$4,end_minutes=$5 WHERE id=$1 AND staff_id=$2`, slotID, staffID, slot.Weekday, slot.StartMinutes, slot.EndMinutes)
			if err != nil {
				return nil, err
			}
			if command.RowsAffected() == 0 {
				return nil, application.ErrValidation
			}
		} else if err := tx.QueryRow(ctx, `INSERT INTO service_staff_weekly_availability(staff_id,weekday,start_minutes,end_minutes)VALUES($1,$2,$3,$4) RETURNING id`, staffID, slot.Weekday, slot.StartMinutes, slot.EndMinutes).Scan(&slotID); err != nil {
			return nil, err
		}
		ids, kept = append(ids, slotID), append(kept, slotID)
	}
	// Availability rows are not referenced by appointments; appointments retain
	// their scheduled range and staff reference, so obsolete windows can be removed.
	_, err := tx.Exec(ctx, `DELETE FROM service_staff_weekly_availability WHERE staff_id=$1 AND id <> ALL($2::uuid[])`, staffID, kept)
	return ids, err
}

func persistedID(value string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	return id, err == nil
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func floatValue(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}
func (r *CommandRepository) UpdateOwned(ctx context.Context, a application.Actor, id uuid.UUID, i domain.UpdateInput) (domain.Business, error) {
	var business domain.Business
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		updated, err := scan(tx.QueryRow(ctx, "UPDATE businesses SET name=COALESCE($3,name),name_normalized=COALESCE($4,name_normalized),category_id=COALESCE($5,category_id),short_description=COALESCE($6,short_description),status=COALESCE($7,status) WHERE id=$1 AND owner_id=$2 AND deleted_at IS NULL RETURNING "+cols, id, a.ID, i.Name, domain.NormalizeOptionalSearchText(i.Name), i.CategoryID, i.ShortDescription, i.Status))
		business = updated
		if err != nil {
			return err
		}
		return r.writeAudit(ctx, tx, business.ID, a.ID, "business_updated")
	})
	return business, err
}
func (r *CommandRepository) ArchiveOwned(ctx context.Context, a application.Actor, id uuid.UUID) error {
	return database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, "UPDATE businesses SET status='archived',deleted_at=now() WHERE id=$1 AND owner_id=$2 AND deleted_at IS NULL", id, a.ID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return application.ErrBusinessNotFound
		}
		return r.writeAudit(ctx, tx, id, a.ID, "business_archived")
	})
}

func (r *CommandRepository) writeAudit(ctx context.Context, tx pgx.Tx, businessID uuid.UUID, actorUserID, action string) error {
	return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorUserID, Action: action})
}
func (r *CommandRepository) SetSelectedBusiness(ctx context.Context, a application.Actor, id uuid.UUID) error {
	q, e := r.pool.Exec(ctx, "UPDATE users SET selected_business_id=$2 WHERE id=$1 AND EXISTS(SELECT 1 FROM businesses WHERE id=$2 AND owner_id=$1 AND deleted_at IS NULL AND status!='archived')", a.ID, id)
	if e != nil {
		return e
	}
	if q.RowsAffected() == 0 {
		return application.ErrBusinessNotFound
	}
	return nil
}

type s interface{ Scan(...any) error }

func scan(row s) (domain.Business, error) {
	var b domain.Business
	e := row.Scan(&b.ID, &b.OwnerID, &b.Type, &b.Status, &b.Name, &b.CategoryID, &b.Currency, &b.ShortDescription, &b.AverageRating, &b.ReviewCount, &b.IsPromotionActive, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return b, application.ErrBusinessNotFound
	}
	return b, e
}
