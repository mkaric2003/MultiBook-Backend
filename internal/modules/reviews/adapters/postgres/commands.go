package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.Repository = (*CommandRepository)(nil)

type sourceSnapshot struct {
	businessID      uuid.UUID
	businessOwnerID string
	customerID      string
	customerName    string
	avatarPath      *string
	finished        bool
}

func (r *CommandRepository) Create(ctx context.Context, customerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Review, error) {
	var review domain.Review
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var businessMarker int
		err := tx.QueryRow(ctx, `SELECT 1 FROM businesses WHERE id=$1 FOR UPDATE`, businessID).Scan(&businessMarker)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load review business: %w", err)
		}

		source, err := loadSource(ctx, tx, input.Type, input.SourceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load review source: %w", err)
		}
		var alreadyReviewed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM business_reviews WHERE business_id=$1 AND customer_id=$2)`, businessID, customerID).Scan(&alreadyReviewed); err != nil {
			return fmt.Errorf("check existing review: %w", err)
		}
		if alreadyReviewed {
			return application.ErrAlreadyReviewed
		}
		if source.customerID != customerID || source.businessID != businessID {
			return application.ErrForbidden
		}
		if !source.finished {
			return application.ErrSourceNotFinished
		}

		review = domain.Review{
			ID: uuid.New(), BusinessID: businessID, BusinessOwnerID: source.businessOwnerID,
			CustomerID: customerID, CustomerName: source.customerName, CustomerAvatarPath: source.avatarPath,
			SourceID: input.SourceID, SourceType: input.Type, Rating: input.Rating, Comment: input.Comment,
		}
		var stayBookingID, serviceAppointmentID *uuid.UUID
		if input.Type == domain.SourceTypeStay {
			stayBookingID = &input.SourceID
		} else {
			serviceAppointmentID = &input.SourceID
		}
		err = tx.QueryRow(ctx, `INSERT INTO business_reviews(
			id,business_id,business_owner_id,customer_id,customer_name,customer_avatar_path,
			stay_booking_id,service_appointment_id,rating,comment)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING created_at`,
			review.ID, review.BusinessID, review.BusinessOwnerID, review.CustomerID, review.CustomerName,
			review.CustomerAvatarPath, stayBookingID, serviceAppointmentID, review.Rating, review.Comment,
		).Scan(&review.CreatedAt)
		if err != nil {
			return mapCreateError(err)
		}
		_, err = tx.Exec(ctx, `UPDATE businesses
			SET average_rating=round(((average_rating*review_count)+$2::numeric)/(review_count+1),2),
				review_count=review_count+1
			WHERE id=$1`, businessID, input.Rating)
		return err
	})
	if err != nil {
		return domain.Review{}, err
	}
	return review, nil
}

func loadSource(ctx context.Context, tx pgx.Tx, sourceType domain.SourceType, sourceID uuid.UUID) (sourceSnapshot, error) {
	var source sourceSnapshot
	var row pgx.Row
	if sourceType == domain.SourceTypeStay {
		row = tx.QueryRow(ctx, `SELECT s.business_id,s.business_owner_id,s.customer_id,s.customer_name,
			COALESCE(s.customer_avatar_path,u.avatar_storage_path),
			(s.status='completed' OR s.check_out < current_date)
			FROM stay_bookings s JOIN users u ON u.id=s.customer_id WHERE s.id=$1`, sourceID)
	} else {
		row = tx.QueryRow(ctx, `SELECT a.business_id,b.owner_id,a.customer_id,a.customer_name,
			COALESCE(a.customer_avatar_path,u.avatar_storage_path),
			(a.status='completed' OR upper(a.scheduled_range) <= now())
			FROM service_appointments a JOIN businesses b ON b.id=a.business_id
			JOIN users u ON u.id=a.customer_id WHERE a.id=$1`, sourceID)
	}
	err := row.Scan(&source.businessID, &source.businessOwnerID, &source.customerID, &source.customerName, &source.avatarPath, &source.finished)
	return source, err
}

func mapCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return application.ErrAlreadyReviewed
	}
	return err
}
