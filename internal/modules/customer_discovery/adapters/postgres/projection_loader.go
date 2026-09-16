package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

func (q *Queries) loadMedia(ctx context.Context, base *application.CustomerBusinessBase, includeCollections bool) error {
	rows, err := q.pool.Query(ctx, `SELECT media_type::text, storage_path FROM business_media
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
	collections, err := q.pool.Query(ctx, `SELECT collection_id FROM business_featured_collections WHERE business_id=$1 ORDER BY collection_id`, base.ID)
	if err != nil {
		return err
	}
	defer collections.Close()
	for collections.Next() {
		var id string
		if err := collections.Scan(&id); err != nil {
			return err
		}
		base.FeaturedCollectionIDs = append(base.FeaturedCollectionIDs, id)
	}
	return collections.Err()
}

func (q *Queries) loadStaySummary(ctx context.Context, businessID uuid.UUID) (application.StaySummary, error) {
	var stay application.StaySummary
	err := q.pool.QueryRow(ctx, `SELECT inventory_type::text, COALESCE(base_price_minor,
		(SELECT MIN(price_per_night_minor) FROM stay_unit_types WHERE business_id=$1 AND deleted_at IS NULL AND is_active))
		FROM stay_details WHERE business_id=$1`, businessID).Scan(&stay.InventoryType, &stay.PricePerNight)
	return stay, err
}

func (q *Queries) loadStayDetail(ctx context.Context, businessID uuid.UUID) (application.StayDetail, error) {
	summary, err := q.loadStaySummary(ctx, businessID)
	if err != nil {
		return application.StayDetail{}, err
	}
	stay := application.StayDetail{StaySummary: summary, Amenities: []string{}, Extras: []application.StayExtra{}, Rooms: []application.StayRoom{}}
	amenities, err := q.pool.Query(ctx, `SELECT amenity_code FROM stay_amenities WHERE business_id=$1 ORDER BY amenity_code`, businessID)
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
	extras, err := q.pool.Query(ctx, `SELECT extra_type, price_minor, pricing_unit FROM stay_extras WHERE business_id=$1 ORDER BY extra_type`, businessID)
	if err != nil {
		return stay, err
	}
	defer extras.Close()
	for extras.Next() {
		var item application.StayExtra
		if err := extras.Scan(&item.Type, &item.Price, &item.PricingUnit); err != nil {
			return stay, err
		}
		stay.Extras = append(stay.Extras, item)
	}
	if err := extras.Err(); err != nil {
		return stay, err
	}
	rooms, err := q.pool.Query(ctx, `SELECT id, name, max_guests, size_square_meters, price_per_night_minor, quantity, is_active FROM stay_unit_types WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, businessID)
	if err != nil {
		return stay, err
	}
	defer rooms.Close()
	for rooms.Next() {
		var item application.StayRoom
		if err := rooms.Scan(&item.ID, &item.Name, &item.MaxGuests, &item.SizeSquareMeters, &item.PricePerNight, &item.Quantity, &item.IsActive); err != nil {
			return stay, err
		}
		stay.Rooms = append(stay.Rooms, item)
	}
	return stay, rooms.Err()
}

func (q *Queries) loadServiceDetail(ctx context.Context, businessID uuid.UUID) (application.ServiceDetail, error) {
	service := application.ServiceDetail{Offerings: []application.ServiceOffering{}, Providers: []application.ServiceProvider{}}
	offerings, err := q.pool.Query(ctx, `SELECT id, name, description, duration_minutes, price_minor, is_active FROM service_offerings WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, businessID)
	if err != nil {
		return service, err
	}
	defer offerings.Close()
	for offerings.Next() {
		var item application.ServiceOffering
		if err := offerings.Scan(&item.ID, &item.Name, &item.Description, &item.DurationMinutes, &item.Price, &item.IsActive); err != nil {
			return service, err
		}
		service.Offerings = append(service.Offerings, item)
	}
	if err := offerings.Err(); err != nil {
		return service, err
	}
	providers, err := q.pool.Query(ctx, `SELECT id, name, title, commission_rate, is_active FROM service_staff WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, businessID)
	if err != nil {
		return service, err
	}
	defer providers.Close()
	for providers.Next() {
		var provider application.ServiceProvider
		if err := providers.Scan(&provider.ID, &provider.Name, &provider.Title, &provider.CommissionRate, &provider.IsActive); err != nil {
			return service, err
		}
		availability, err := q.pool.Query(ctx, `SELECT id, weekday, start_minutes, end_minutes FROM service_staff_weekly_availability WHERE staff_id=$1 ORDER BY weekday, start_minutes`, provider.ID)
		if err != nil {
			return service, err
		}
		for availability.Next() {
			var slot application.WeeklyAvailability
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
			provider.Availability = []application.WeeklyAvailability{}
		}
		service.Providers = append(service.Providers, provider)
	}
	return service, providers.Err()
}

func (q *Queries) loadServiceSummary(ctx context.Context, businessID uuid.UUID) (application.ServiceSummary, error) {
	service := application.ServiceSummary{Offerings: []application.ServiceOffering{}}
	rows, err := q.pool.Query(ctx, `SELECT id, name, description, duration_minutes, price_minor, is_active
		FROM service_offerings WHERE business_id=$1 AND is_active AND deleted_at IS NULL
		ORDER BY price_minor, created_at LIMIT 1`, businessID)
	if err != nil {
		return service, err
	}
	defer rows.Close()
	for rows.Next() {
		var item application.ServiceOffering
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.DurationMinutes, &item.Price, &item.IsActive); err != nil {
			return service, err
		}
		service.Offerings = append(service.Offerings, item)
	}
	return service, rows.Err()
}
