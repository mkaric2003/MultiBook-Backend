package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_search/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/stay_search/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

// Search keeps filtering in PostgreSQL, including the date-overlap availability
// check that previously ran in the Firebase callable function.
func (q *Queries) Search(ctx context.Context, in domain.Input) ([]domain.Item, error) {
	query, args := buildSearchQuery(in)
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search stays: %w", err)
	}
	defer rows.Close()
	items := []domain.Item{}
	for rows.Next() {
		var x domain.Item
		if err := rows.Scan(&x.ID, &x.OwnerID, &x.Name, &x.CategoryID, &x.Currency, &x.City, &x.Address, &x.Latitude, &x.Longitude, &x.AverageRating, &x.ReviewCount, &x.ShortDescription, &x.LogoURL, &x.CoverPhotoURL, &x.PhotoURLs, &x.FeaturedCollectionIDs, &x.Amenities, &x.PricePerNight, &x.InventoryType); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func buildSearchQuery(in domain.Input) (string, []any) {
	availabilityClause := ""
	args := []any{in.City, in.CategoryIDs, in.CollectionIDs, in.InventoryType, in.MinimumRating, in.Amenities, in.MinPriceMinor, in.MaxPriceMinor}
	limitParameter, offsetParameter := 9, 10
	if in.CheckIn != "" && in.CheckOut != "" {
		availabilityClause = `
  AND ((c.inventory_type='single_unit' AND NOT EXISTS (
    SELECT 1 FROM stay_bookings sb WHERE sb.business_id=c.id AND sb.status='confirmed' AND sb.check_in < $11::date AND sb.check_out > $10::date
  )) OR (c.inventory_type='multiple_units' AND EXISTS (
    SELECT 1 FROM stay_unit_types t WHERE t.business_id=c.id AND t.is_active AND t.deleted_at IS NULL AND t.max_guests >= $9
    AND (SELECT count(*) FROM stay_bookings sb WHERE sb.business_id=c.id AND sb.stay_unit_type_id=t.id AND sb.status='confirmed' AND sb.check_in < $11::date AND sb.check_out > $10::date) < t.quantity
  )))`
		args = append(args, in.Adults+in.Children, in.CheckIn, in.CheckOut)
		limitParameter, offsetParameter = 12, 13
	}
	args = append(args, in.PageSize, in.Offset)
	query := fmt.Sprintf(`
WITH candidates AS (
 SELECT b.id,b.owner_id,b.name,b.category_id,b.currency,b.short_description,b.average_rating,b.review_count,
        COALESCE(l.city, '') AS city,COALESCE(l.address, '') AS address,COALESCE(ST_Y(l.coordinates::geometry), 0) AS latitude,COALESCE(ST_X(l.coordinates::geometry), 0) AS longitude,
        sd.inventory_type::text, COALESCE(sd.base_price_minor,(SELECT MIN(sut.price_per_night_minor) FROM stay_unit_types sut WHERE sut.business_id=b.id AND sut.is_active AND sut.deleted_at IS NULL)) price
 FROM businesses b LEFT JOIN business_locations l ON l.business_id=b.id AND l.is_primary JOIN stay_details sd ON sd.business_id=b.id
 WHERE b.type='stay' AND b.status='active' AND b.deleted_at IS NULL
   AND ($1='' OR l.city_normalized=$1)
   AND (COALESCE(cardinality($2::text[]), 0)=0 OR b.category_id=ANY($2))
   AND (COALESCE(cardinality($3::text[]), 0)=0 OR EXISTS (SELECT 1 FROM business_featured_collections c WHERE c.business_id=b.id AND c.collection_id=ANY($3)))
   AND ($4='' OR sd.inventory_type::text=CASE WHEN $4='singleUnit' THEN 'single_unit' ELSE 'multiple_units' END)
   AND b.average_rating >= $5
   AND (COALESCE(cardinality($6::text[]), 0)=0 OR (SELECT count(*) FROM stay_amenities a WHERE a.business_id=b.id AND a.amenity_code=ANY($6))=cardinality($6::text[]))
)
SELECT c.id,c.owner_id,c.name,c.category_id,c.currency,c.city,c.address,c.latitude,c.longitude,c.average_rating,c.review_count,c.short_description,
       (SELECT storage_path FROM business_media WHERE business_id=c.id AND media_type='logo'),
       (SELECT storage_path FROM business_media WHERE business_id=c.id AND media_type='cover'),
       COALESCE((SELECT array_agg(storage_path ORDER BY position) FROM business_media WHERE business_id=c.id AND media_type='gallery'),ARRAY[]::text[]),
       COALESCE((SELECT array_agg(collection_id ORDER BY collection_id) FROM business_featured_collections WHERE business_id=c.id),ARRAY[]::text[]),
       COALESCE((SELECT array_agg(amenity_code ORDER BY amenity_code) FROM stay_amenities WHERE business_id=c.id),ARRAY[]::text[]),c.price,c.inventory_type
FROM candidates c
WHERE c.price BETWEEN $7 AND $8
	%s
ORDER BY c.average_rating DESC,c.review_count DESC,c.name,c.id LIMIT $%d OFFSET $%d`, availabilityClause, limitParameter, offsetParameter)
	return query, args
}
