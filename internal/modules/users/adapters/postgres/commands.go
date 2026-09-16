// Package postgres is the PostgreSQL command adapter for the users module.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

// CommandRepository is the Postgres write-side adapter for users.
type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.UserRepository = (*CommandRepository)(nil)

func (r *CommandRepository) ProvisionFromAuthentication(ctx context.Context, input application.AuthenticatedUser) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, NULLIF($2, ''), NULLIF($3, '')) ON CONFLICT (id) DO NOTHING`, input.ID, input.Email, input.DisplayName)
	if err != nil {
		return fmt.Errorf("provision authenticated user: %w", err)
	}
	return nil
}

func (r *CommandRepository) SetRole(ctx context.Context, id string, role domain.UserRole) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `UPDATE users SET role = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING `+userColumns, id, role))
}

func (r *CommandRepository) UpdateProfile(ctx context.Context, id string, input domain.UpdateProfileInput) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `UPDATE users SET first_name = COALESCE($2, first_name), last_name = COALESCE($3, last_name), display_name = CASE WHEN $2 IS NOT NULL OR $3 IS NOT NULL THEN NULLIF(BTRIM(CONCAT_WS(' ', COALESCE($2, first_name), COALESCE($3, last_name))), '') ELSE display_name END, phone_e164 = COALESCE($4, phone_e164), avatar_storage_path = COALESCE($5, avatar_storage_path), country_code = COALESCE($6, country_code), date_of_birth = COALESCE($7, date_of_birth), address = COALESCE($8, address), city = COALESCE($9, city), city_normalized = COALESCE($10, city_normalized), business_currency = COALESCE($11, business_currency) WHERE id = $1 AND deleted_at IS NULL RETURNING `+userColumns, id, input.FirstName, input.LastName, input.PhoneE164, input.AvatarStoragePath, input.CountryCode, input.DateOfBirth, input.Address, input.City, normalized(input.City), input.BusinessCurrency))
}

func normalized(value *string) *string {
	if value == nil {
		return nil
	}
	result := strings.ToLower(strings.TrimSpace(*value))
	return &result
}
