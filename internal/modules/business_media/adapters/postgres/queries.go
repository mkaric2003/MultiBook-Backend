package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/business_media/domain"
)

func list(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID uuid.UUID) ([]domain.Media, error) {
	rows, err := db.Query(ctx, "SELECT id,media_type,storage_path,position FROM business_media WHERE business_id=$1 ORDER BY CASE media_type WHEN 'logo' THEN 0 WHEN 'cover' THEN 1 ELSE 2 END,position", businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Media, 0)
	for rows.Next() {
		var item domain.Media
		if err = rows.Scan(&item.ID, &item.MediaType, &item.StoragePath, &item.Position); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
