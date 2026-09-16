package application

import "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"

type ListInput struct {
	Offset int
	Limit  int
}

type Page struct {
	Items      []domain.Notification
	NextCursor *string
}
