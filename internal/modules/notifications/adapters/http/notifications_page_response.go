package http

import "github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"

func notificationsPageResponse(page application.Page) map[string]any {
	items := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, notificationResponse(item))
	}
	return map[string]any{"items": items, "nextCursor": page.NextCursor}
}
