package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

// Queries implements the customer discovery read port with active-only
// PostgreSQL projections.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)
var _ application.CustomerBusinessSummaryReader = (*Queries)(nil)

const baseColumns = `b.id, b.owner_id, b.type, b.status, b.name, b.category_id,
	b.currency, b.short_description, b.average_rating, b.review_count,
	b.is_promotion_active, b.created_at, b.updated_at, l.city, l.address,
	ST_Y(l.coordinates::geometry), ST_X(l.coordinates::geometry)`

func (q *Queries) GetActiveDetail(ctx context.Context, id uuid.UUID) (application.CustomerBusinessDetail, error) {
	base, err := q.readBase(ctx, `SELECT `+baseColumns+`
		FROM businesses b JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		WHERE b.id=$1 AND b.status='active' AND b.deleted_at IS NULL`, id)
	if err != nil {
		return application.CustomerBusinessDetail{}, err
	}
	if err := q.loadMedia(ctx, &base, true); err != nil {
		return application.CustomerBusinessDetail{}, err
	}
	detail := application.CustomerBusinessDetail{CustomerBusinessBase: base}
	if base.Type == domain.BusinessTypeStay {
		stay, err := q.loadStayDetail(ctx, base.ID)
		if err != nil {
			return application.CustomerBusinessDetail{}, err
		}
		detail.Stay = &stay
	} else {
		service, err := q.loadServiceDetail(ctx, base.ID)
		if err != nil {
			return application.CustomerBusinessDetail{}, err
		}
		detail.Service = &service
	}
	return detail, nil
}

func (q *Queries) ListPopular(ctx context.Context, businessType domain.BusinessType, city string, limit, offset int) ([]application.CustomerBusinessSummary, error) {
	bases, err := q.listBases(ctx, `SELECT `+baseColumns+`
		FROM businesses b JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		WHERE b.type=$1 AND b.status='active' AND b.deleted_at IS NULL
			AND ($2 = '' OR l.city_normalized=$2)
		ORDER BY b.average_rating DESC, b.review_count DESC, b.name LIMIT $3 OFFSET $4`, businessType, city, limit, offset)
	if err != nil {
		return nil, err
	}
	return q.summaries(ctx, bases)
}

// ListActiveByIDs returns current customer summaries in the same order as the
// supplied IDs. Missing, inactive, archived, and soft-deleted businesses are
// intentionally omitted.
func (q *Queries) ListActiveByIDs(ctx context.Context, ids []uuid.UUID) ([]application.CustomerBusinessSummary, error) {
	if len(ids) == 0 {
		return []application.CustomerBusinessSummary{}, nil
	}
	bases, err := q.listBases(ctx, `SELECT `+baseColumns+`
		FROM unnest($1::uuid[]) WITH ORDINALITY requested(id, position)
		JOIN businesses b ON b.id=requested.id
		JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		WHERE b.status='active' AND b.deleted_at IS NULL
		ORDER BY requested.position`, ids)
	if err != nil {
		return nil, err
	}
	return q.summaries(ctx, bases)
}

func (q *Queries) ListCities(ctx context.Context) ([]string, error) {
	rows, err := q.pool.Query(ctx, `SELECT name FROM cities ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cities := []string{}
	for rows.Next() {
		var city string
		if err := rows.Scan(&city); err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}
	return cities, rows.Err()
}

func (q *Queries) ListFeaturedCollections(ctx context.Context, businessType domain.BusinessType) ([]application.FeaturedCollection, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,title_key,subtitle_key,image_url FROM featured_collections WHERE business_type=$1 ORDER BY position`, businessType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []application.FeaturedCollection{}
	for rows.Next() {
		var item application.FeaturedCollection
		if err := rows.Scan(&item.ID, &item.TitleKey, &item.SubtitleKey, &item.ImageURL); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) Search(ctx context.Context, businessType domain.BusinessType, query string, limit int) ([]application.CustomerBusinessSummary, error) {
	bases, err := q.listBases(ctx, `SELECT `+baseColumns+`
		FROM businesses b JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		WHERE b.type=$1 AND b.status='active' AND b.deleted_at IS NULL
			AND (b.name_normalized LIKE $2 || '%' OR l.city_normalized LIKE $2 || '%')
		ORDER BY b.average_rating DESC, b.review_count DESC, b.name LIMIT $3`, businessType, query, limit)
	if err != nil {
		return nil, err
	}
	return q.summaries(ctx, bases)
}

func (q *Queries) ListRecommendedStays(ctx context.Context, city string, limit int) ([]application.RecommendedStay, error) {
	bases, err := q.listBases(ctx, `SELECT `+baseColumns+`
		FROM businesses b JOIN business_locations l ON l.business_id=b.id AND l.is_primary
		JOIN stay_details sd ON sd.business_id=b.id
		WHERE b.type='stay' AND b.status='active' AND b.deleted_at IS NULL
		ORDER BY CASE WHEN $1 <> '' AND l.city_normalized=$1 THEN 0 ELSE 1 END,
		b.average_rating DESC, b.review_count DESC, b.name LIMIT $2`, city, limit)
	if err != nil {
		return nil, err
	}
	items := make([]application.RecommendedStay, 0, len(bases))
	for _, base := range bases {
		if err := q.loadMedia(ctx, &base, false); err != nil {
			return nil, err
		}
		stay, err := q.loadStaySummary(ctx, base.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, application.RecommendedStay{CustomerBusinessBase: base, Stay: stay})
	}
	return items, nil
}

func (q *Queries) summaries(ctx context.Context, bases []application.CustomerBusinessBase) ([]application.CustomerBusinessSummary, error) {
	items := make([]application.CustomerBusinessSummary, 0, len(bases))
	for _, base := range bases {
		if err := q.loadMedia(ctx, &base, false); err != nil {
			return nil, err
		}
		item := application.CustomerBusinessSummary{CustomerBusinessBase: base}
		if base.Type == domain.BusinessTypeStay {
			stay, err := q.loadStaySummary(ctx, base.ID)
			if err != nil {
				return nil, err
			}
			item.Stay = &stay
		} else {
			service, err := q.loadServiceSummary(ctx, base.ID)
			if err != nil {
				return nil, err
			}
			item.Service = &service
		}
		items = append(items, item)
	}
	return items, nil
}

func (q *Queries) listBases(ctx context.Context, query string, args ...any) ([]application.CustomerBusinessBase, error) {
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []application.CustomerBusinessBase{}
	for rows.Next() {
		item, err := scanBase(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) readBase(ctx context.Context, query string, args ...any) (application.CustomerBusinessBase, error) {
	item, err := scanBase(q.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.CustomerBusinessBase{}, application.ErrBusinessNotFound
	}
	return item, err
}

type scanner interface{ Scan(...any) error }

func scanBase(row scanner) (application.CustomerBusinessBase, error) {
	var item application.CustomerBusinessBase
	err := row.Scan(&item.ID, &item.OwnerID, &item.Type, &item.Status, &item.Name, &item.CategoryID,
		&item.Currency, &item.ShortDescription, &item.AverageRating, &item.ReviewCount,
		&item.IsPromotionActive, &item.CreatedAt, &item.UpdatedAt, &item.Location.City,
		&item.Location.Address, &item.Location.Latitude, &item.Location.Longitude)
	return item, err
}
