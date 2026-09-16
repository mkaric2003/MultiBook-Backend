package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_locations/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_locations/domain"
	businessdomain "github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

// CommandRepository is the Postgres write-side adapter for business locations.
type CommandRepository struct {
	pool  *pgxpool.Pool
	audit auditapp.Writer
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit}
}

var _ application.LocationRepository = (*CommandRepository)(nil)

func (r *CommandRepository) UpsertOwned(ctx context.Context, actorID string, businessID uuid.UUID, input domain.UpsertInput) (domain.Location, error) {
	var location domain.Location
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO business_locations (business_id, city, city_normalized, address, country_code, coordinates, is_primary) SELECT $1,$3,$4,$5,$6,ST_SetSRID(ST_MakePoint($7,$8),4326)::geography,TRUE WHERE EXISTS (SELECT 1 FROM businesses WHERE id=$1 AND owner_id=$2 AND deleted_at IS NULL) ON CONFLICT (business_id) WHERE is_primary DO UPDATE SET city=EXCLUDED.city,city_normalized=EXCLUDED.city_normalized,address=EXCLUDED.address,country_code=EXCLUDED.country_code,coordinates=EXCLUDED.coordinates RETURNING id,city,address,country_code,ST_Y(coordinates::geometry),ST_X(coordinates::geometry)`, businessID, actorID, input.City, businessdomain.NormalizeSearchText(input.City), input.Address, input.CountryCode, input.Longitude, input.Latitude).Scan(&location.ID, &location.City, &location.Address, &location.CountryCode, &location.Latitude, &location.Longitude)
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: actorID, Action: "business_location_updated"})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Location{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Location{}, fmt.Errorf("upsert business location: %w", err)
	}
	return location, nil
}
