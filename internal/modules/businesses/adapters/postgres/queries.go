package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// Queries is the Postgres read-side adapter for business list and detail
// screens. SQL implementations are intentionally typed and transport-agnostic.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.BusinessQueries = (*Queries)(nil)

func (r *Queries) ListOwned(ctx context.Context, owner string) ([]application.OwnedSummary, error) {
	return r.listOwned(ctx, owner)
}

func (r *Queries) GetOwnedDetail(ctx context.Context, owner string, id uuid.UUID) (application.OwnedDetail, error) {
	return r.getOwnedDetail(ctx, owner, id)
}

const readBaseColumns = `b.id, b.owner_id, b.type, b.status, b.name, b.category_id,
	b.currency, b.short_description, b.average_rating, b.review_count,
	b.is_promotion_active, b.created_at, b.updated_at, l.city, l.address,
	ST_Y(l.coordinates::geometry), ST_X(l.coordinates::geometry)`

func (r *Queries) listOwned(ctx context.Context, owner string) ([]application.OwnedSummary, error) {
	bases, err := r.listReadBases(ctx, `SELECT `+readBaseColumns+`
		FROM businesses b JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		WHERE b.owner_id=$1 AND b.deleted_at IS NULL ORDER BY b.created_at DESC`, owner)
	if err != nil {
		return nil, err
	}
	items := make([]application.OwnedSummary, 0, len(bases))
	for _, base := range bases {
		if err := r.loadMedia(ctx, &base, true); err != nil {
			return nil, err
		}
		item := application.OwnedSummary{BusinessReadBase: base}
		if base.Type == domain.BusinessTypeStay {
			stay, err := r.loadStaySummary(ctx, base.ID)
			if err != nil {
				return nil, err
			}
			item.Stay = &stay
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Queries) getOwnedDetail(ctx context.Context, owner string, id uuid.UUID) (application.OwnedDetail, error) {
	base, err := r.readBase(ctx, `SELECT `+readBaseColumns+`
		FROM businesses b JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		WHERE b.id=$1 AND b.owner_id=$2 AND b.deleted_at IS NULL`, id, owner)
	if err != nil {
		return application.OwnedDetail{}, err
	}
	return r.detailForBase(ctx, base)
}

func (r *Queries) detailForBase(ctx context.Context, base application.BusinessReadBase) (application.OwnedDetail, error) {
	if err := r.loadMedia(ctx, &base, true); err != nil {
		return application.OwnedDetail{}, err
	}
	item := application.OwnedDetail{BusinessReadBase: base}
	if base.Type == domain.BusinessTypeStay {
		stay, err := r.loadStayDetail(ctx, base.ID)
		if err != nil {
			return application.OwnedDetail{}, err
		}
		item.Stay = &stay
	} else {
		service, err := r.loadServiceDetail(ctx, base.ID)
		if err != nil {
			return application.OwnedDetail{}, err
		}
		item.Service = &service
	}
	return item, nil
}

func (r *Queries) listReadBases(ctx context.Context, query string, args ...any) ([]application.BusinessReadBase, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []application.BusinessReadBase{}
	for rows.Next() {
		item, err := scanReadBase(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Queries) readBase(ctx context.Context, query string, args ...any) (application.BusinessReadBase, error) {
	item, err := scanReadBase(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.BusinessReadBase{}, application.ErrBusinessNotFound
	}
	return item, err
}

func scanReadBase(row s) (application.BusinessReadBase, error) {
	var item application.BusinessReadBase
	err := row.Scan(&item.ID, &item.OwnerID, &item.Type, &item.Status, &item.Name, &item.CategoryID,
		&item.Currency, &item.ShortDescription, &item.AverageRating, &item.ReviewCount,
		&item.IsPromotionActive, &item.CreatedAt, &item.UpdatedAt, &item.Location.City,
		&item.Location.Address, &item.Location.Latitude, &item.Location.Longitude)
	return item, err
}

func (r *Queries) loadMedia(ctx context.Context, base *application.BusinessReadBase, includeCollections bool) error {
	rows, err := r.pool.Query(ctx, `SELECT media_type::text, storage_path FROM business_media
		WHERE business_id=$1 ORDER BY CASE media_type WHEN 'gallery' THEN 1 ELSE 0 END, position`, base.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	base.PhotoPaths = []string{}
	for rows.Next() {
		var mediaType, path string
		if err := rows.Scan(&mediaType, &path); err != nil {
			return err
		}
		switch mediaType {
		case "logo":
			base.LogoPath = &path
		case "cover":
			base.CoverPath = &path
		case "gallery":
			base.PhotoPaths = append(base.PhotoPaths, path)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	base.FeaturedCollectionIDs = []string{}
	if !includeCollections {
		return nil
	}
	collections, err := r.pool.Query(ctx, `SELECT collection_id FROM business_featured_collections WHERE business_id=$1 ORDER BY collection_id`, base.ID)
	if err != nil {
		return err
	}
	defer collections.Close()
	for collections.Next() {
		var collectionID string
		if err := collections.Scan(&collectionID); err != nil {
			return err
		}
		base.FeaturedCollectionIDs = append(base.FeaturedCollectionIDs, collectionID)
	}
	return collections.Err()
}

func (r *Queries) loadStaySummary(ctx context.Context, businessID uuid.UUID) (application.StaySummaryRead, error) {
	var stay application.StaySummaryRead
	err := r.pool.QueryRow(ctx, `SELECT inventory_type::text, COALESCE(base_price_minor,
		(SELECT MIN(price_per_night_minor) FROM stay_unit_types WHERE business_id=$1 AND deleted_at IS NULL AND is_active))
		FROM stay_details WHERE business_id=$1`, businessID).Scan(&stay.InventoryType, &stay.PricePerNight)
	return stay, err
}

func (r *Queries) loadStayDetail(ctx context.Context, businessID uuid.UUID) (application.StayDetailRead, error) {
	summary, err := r.loadStaySummary(ctx, businessID)
	if err != nil {
		return application.StayDetailRead{}, err
	}
	stay := application.StayDetailRead{StaySummaryRead: summary, Amenities: []string{}, Extras: []application.StayExtraRead{}, Rooms: []application.StayRoomRead{}}
	amenities, err := r.pool.Query(ctx, `SELECT amenity_code FROM stay_amenities WHERE business_id=$1 ORDER BY amenity_code`, businessID)
	if err != nil {
		return stay, err
	}
	defer amenities.Close()
	for amenities.Next() {
		var value string
		if err := amenities.Scan(&value); err != nil {
			return stay, err
		}
		stay.Amenities = append(stay.Amenities, value)
	}
	if err := amenities.Err(); err != nil {
		return stay, err
	}
	extras, err := r.pool.Query(ctx, `SELECT extra_type, price_minor, pricing_unit FROM stay_extras WHERE business_id=$1 ORDER BY extra_type`, businessID)
	if err != nil {
		return stay, err
	}
	defer extras.Close()
	for extras.Next() {
		var item application.StayExtraRead
		if err := extras.Scan(&item.Type, &item.Price, &item.PricingUnit); err != nil {
			return stay, err
		}
		stay.Extras = append(stay.Extras, item)
	}
	if err := extras.Err(); err != nil {
		return stay, err
	}
	rooms, err := r.pool.Query(ctx, `SELECT id, name, max_guests, size_square_meters, price_per_night_minor, quantity, is_active FROM stay_unit_types WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, businessID)
	if err != nil {
		return stay, err
	}
	defer rooms.Close()
	for rooms.Next() {
		var item application.StayRoomRead
		if err := rooms.Scan(&item.ID, &item.Name, &item.MaxGuests, &item.SizeSquareMeters, &item.PricePerNight, &item.Quantity, &item.IsActive); err != nil {
			return stay, err
		}
		stay.Rooms = append(stay.Rooms, item)
	}
	return stay, rooms.Err()
}

func (r *Queries) loadServiceDetail(ctx context.Context, businessID uuid.UUID) (application.ServiceDetailRead, error) {
	service := application.ServiceDetailRead{Offerings: []application.ServiceOfferingRead{}, Providers: []application.ServiceProviderRead{}}
	offerings, err := r.pool.Query(ctx, `SELECT id, name, description, duration_minutes, price_minor, is_active FROM service_offerings WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, businessID)
	if err != nil {
		return service, err
	}
	defer offerings.Close()
	for offerings.Next() {
		var item application.ServiceOfferingRead
		if err := offerings.Scan(&item.ID, &item.Name, &item.Description, &item.DurationMinutes, &item.Price, &item.IsActive); err != nil {
			return service, err
		}
		service.Offerings = append(service.Offerings, item)
	}
	if err := offerings.Err(); err != nil {
		return service, err
	}
	providers, err := r.pool.Query(ctx, `SELECT id, name, title, commission_rate, is_active FROM service_staff WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, businessID)
	if err != nil {
		return service, err
	}
	defer providers.Close()
	for providers.Next() {
		var provider application.ServiceProviderRead
		if err := providers.Scan(&provider.ID, &provider.Name, &provider.Title, &provider.CommissionRate, &provider.IsActive); err != nil {
			return service, err
		}
		availability, err := r.pool.Query(ctx, `SELECT id, weekday, start_minutes, end_minutes FROM service_staff_weekly_availability WHERE staff_id=$1 ORDER BY weekday, start_minutes`, provider.ID)
		if err != nil {
			return service, err
		}
		for availability.Next() {
			var slot application.WeeklyAvailabilityRead
			if err := availability.Scan(&slot.ID, &slot.Weekday, &slot.StartMinutes, &slot.EndMinutes); err != nil {
				availability.Close()
				return service, err
			}
			provider.Availability = append(provider.Availability, slot)
		}
		if err := availability.Err(); err != nil {
			availability.Close()
			return service, err
		}
		availability.Close()
		if provider.Availability == nil {
			provider.Availability = []application.WeeklyAvailabilityRead{}
		}
		service.Providers = append(service.Providers, provider)
	}
	return service, providers.Err()
}
