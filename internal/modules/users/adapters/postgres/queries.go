// Package postgres is the PostgreSQL read adapter for the users module.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

// Queries is the Postgres read-side adapter for users.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.UserQueries = (*Queries)(nil)

const userColumns = `id, email, display_name, first_name, last_name, role, phone_e164,
	avatar_storage_path, country_code, date_of_birth, address, city, business_currency, selected_business_id, created_at, updated_at`

func (r *Queries) GetByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL`, id))
}

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User
	var dateOfBirth *time.Time
	if err := row.Scan(&user.ID, &user.Email, &user.FullName, &user.FirstName, &user.LastName, &user.Role, &user.PhoneE164, &user.AvatarStoragePath, &user.CountryCode, &dateOfBirth, &user.Address, &user.City, &user.BusinessCurrency, &user.SelectedBusinessID, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, application.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}
	if dateOfBirth != nil {
		value := dateOfBirth.Format("2006-01-02")
		user.DateOfBirth = &value
	}
	return user, nil
}
