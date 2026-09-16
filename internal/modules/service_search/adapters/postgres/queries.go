package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_search/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_search/domain"
)

// Queries is the Postgres read-side adapter for service discovery search.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.ServiceSearchQueries = (*Queries)(nil)

// Search returns the complete matching set. This mirrors Firebase's callable:
// availability is evaluated first, then results are sorted and paginated.
func (r *Queries) Search(ctx context.Context, input domain.Input) ([]domain.Item, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.id,b.owner_id,b.name,b.category_id,
		       COALESCE(l.city,''),COALESCE(l.address,''),
		       COALESCE(ST_Y(l.coordinates::geometry),0),COALESCE(ST_X(l.coordinates::geometry),0),
		       b.currency,b.short_description,b.average_rating,b.review_count,
		       b.is_promotion_active,b.created_at,b.updated_at,
		       (SELECT min(price_minor) FROM service_offerings
		          WHERE business_id=b.id AND is_active AND deleted_at IS NULL)
		FROM businesses b
		LEFT JOIN business_locations l ON l.business_id=b.id
		WHERE b.type='service' AND b.status='active' AND b.deleted_at IS NULL
		  AND ($1='' OR lower(translate(l.city,'čćšžđČĆŠŽĐ','ccszdCCSZD'))=$1)
		  AND ($2='' OR b.category_id=$2)
		  AND ($3='' OR EXISTS (
		      SELECT 1 FROM business_featured_collections c
		      WHERE c.business_id=b.id AND c.collection_id=$3
		  ))
		  AND EXISTS (
		      SELECT 1 FROM service_offerings offering
		      WHERE offering.business_id=b.id AND offering.is_active
		        AND offering.deleted_at IS NULL
		        AND offering.price_minor BETWEEN $4 AND $5
		  )
		  AND (NULLIF($6,'') IS NULL OR EXISTS (
		      SELECT 1
		      FROM service_details details
		      JOIN service_staff staff ON staff.business_id=details.business_id
		      JOIN service_staff_offerings staff_offering ON staff_offering.staff_id=staff.id
		      JOIN service_offerings offering ON offering.id=staff_offering.offering_id
		      WHERE staff.is_active AND staff.deleted_at IS NULL
		        AND offering.is_active AND offering.deleted_at IS NULL
		        AND EXISTS (
		          SELECT 1 FROM service_staff_weekly_availability availability
		          WHERE availability.staff_id=staff.id
		            AND availability.weekday=((EXTRACT(DOW FROM NULLIF($6,'')::date)::int+6)%7)
		            AND availability.start_minutes <= $7
		            AND availability.end_minutes >= $7+offering.duration_minutes
		        )
		        AND NOT EXISTS (
		          SELECT 1 FROM service_staff_availability_blocks block
		          WHERE block.staff_id=staff.id
		            AND block.blocked_range && tstzrange(
		              (NULLIF($6,'')::date::timestamp AT TIME ZONE details.time_zone)+$7*interval '1 minute',
		              (NULLIF($6,'')::date::timestamp AT TIME ZONE details.time_zone)+($7+offering.duration_minutes)*interval '1 minute','[)'
		            )
		        )
		        AND NOT EXISTS (
		          SELECT 1 FROM service_appointments appointment
		          WHERE appointment.staff_id=staff.id AND appointment.status='confirmed'
		            AND appointment.scheduled_range && tstzrange(
		              (NULLIF($6,'')::date::timestamp AT TIME ZONE details.time_zone)+$7*interval '1 minute',
		              (NULLIF($6,'')::date::timestamp AT TIME ZONE details.time_zone)+($7+offering.duration_minutes)*interval '1 minute','[)'
		            )
		        )
		        AND details.business_id=b.id
		  ))`, input.City, input.CategoryID, input.CollectionID, input.MinPriceMinor, input.MaxPriceMinor, input.AppointmentDate, input.StartMinutes)
	if err != nil {
		return nil, fmt.Errorf("search services: %w", err)
	}
	defer rows.Close()

	items := []domain.Item{}
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.OwnerID, &item.Name, &item.CategoryID,
			&item.City, &item.Address, &item.Latitude, &item.Longitude,
			&item.Currency, &item.ShortDescription, &item.AverageRating, &item.ReviewCount,
			&item.IsPromotionActive, &item.CreatedAt, &item.UpdatedAt, &item.LowestPriceMinor); err != nil {
			return nil, err
		}
		if err := r.loadDetails(ctx, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Queries) loadDetails(ctx context.Context, item *domain.Item) error {
	media, err := r.pool.Query(ctx, `SELECT media_type::text,storage_path FROM business_media WHERE business_id=$1 ORDER BY media_type,position`, item.ID)
	if err != nil {
		return err
	}
	for media.Next() {
		var kind, path string
		if err := media.Scan(&kind, &path); err != nil {
			media.Close()
			return err
		}
		switch kind {
		case "logo":
			item.LogoPath = &path
		case "cover":
			item.CoverPath = &path
		case "gallery":
			item.PhotoPaths = append(item.PhotoPaths, path)
		}
	}
	if err := media.Err(); err != nil {
		media.Close()
		return err
	}
	media.Close()

	collections, err := r.pool.Query(ctx, `SELECT collection_id FROM business_featured_collections WHERE business_id=$1 ORDER BY collection_id`, item.ID)
	if err != nil {
		return err
	}
	for collections.Next() {
		var id string
		if err := collections.Scan(&id); err != nil {
			collections.Close()
			return err
		}
		item.FeaturedCollectionIDs = append(item.FeaturedCollectionIDs, id)
	}
	if err := collections.Err(); err != nil {
		collections.Close()
		return err
	}
	collections.Close()

	offerings, err := r.pool.Query(ctx, `SELECT id,name,description,duration_minutes,price_minor,is_active FROM service_offerings WHERE business_id=$1 AND is_active AND deleted_at IS NULL ORDER BY price_minor,created_at`, item.ID)
	if err != nil {
		return err
	}
	for offerings.Next() {
		var offering domain.Offering
		if err := offerings.Scan(&offering.ID, &offering.Name, &offering.Description, &offering.DurationMinutes, &offering.PriceMinor, &offering.IsActive); err != nil {
			offerings.Close()
			return err
		}
		item.Offerings = append(item.Offerings, offering)
	}
	if err := offerings.Err(); err != nil {
		offerings.Close()
		return err
	}
	offerings.Close()

	providers, err := r.pool.Query(ctx, `SELECT id,name,title,commission_rate,is_active FROM service_staff WHERE business_id=$1 AND deleted_at IS NULL ORDER BY created_at`, item.ID)
	if err != nil {
		return err
	}
	for providers.Next() {
		var provider domain.Provider
		if err := providers.Scan(&provider.ID, &provider.Name, &provider.Title, &provider.CommissionRate, &provider.IsActive); err != nil {
			providers.Close()
			return err
		}
		availability, err := r.pool.Query(ctx, `SELECT id,weekday,start_minutes,end_minutes FROM service_staff_weekly_availability WHERE staff_id=$1 ORDER BY weekday,start_minutes`, provider.ID)
		if err != nil {
			providers.Close()
			return err
		}
		for availability.Next() {
			var slot domain.AvailabilitySlot
			if err := availability.Scan(&slot.ID, &slot.Weekday, &slot.StartMinutes, &slot.EndMinutes); err != nil {
				availability.Close()
				providers.Close()
				return err
			}
			provider.Availability = append(provider.Availability, slot)
		}
		if err := availability.Err(); err != nil {
			availability.Close()
			providers.Close()
			return err
		}
		availability.Close()
		item.Providers = append(item.Providers, provider)
	}
	if err := providers.Err(); err != nil {
		providers.Close()
		return err
	}
	providers.Close()
	if item.PhotoPaths == nil {
		item.PhotoPaths = []string{}
	}
	if item.FeaturedCollectionIDs == nil {
		item.FeaturedCollectionIDs = []string{}
	}
	if item.Offerings == nil {
		item.Offerings = []domain.Offering{}
	}
	if item.Providers == nil {
		item.Providers = []domain.Provider{}
	}
	return nil
}
