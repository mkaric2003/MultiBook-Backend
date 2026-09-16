package http

import "github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"

func notificationResponse(notification domain.Notification) map[string]any {
	return map[string]any{"id": notification.ID, "kind": notification.Kind, "title": notification.Title, "body": notification.Body, "data": notification.Data, "createdAt": notification.CreatedAt, "readAt": notification.ReadAt}
}
