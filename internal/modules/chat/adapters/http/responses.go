package http

import (
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
)

func conversationResponse(item domain.Conversation) map[string]any {
	participantIDs := []string{item.CustomerID, item.BusinessOwnerID}
	return map[string]any{
		"id": item.ID, "businessId": item.BusinessID, "legacyBusinessId": item.LegacyBusinessID,
		"businessOwnerId": item.BusinessOwnerID, "customerId": item.CustomerID,
		"businessName": item.BusinessName, "businessImageUrl": item.BusinessImagePath,
		"customerName": item.CustomerName, "customerImageUrl": item.CustomerImagePath,
		"participantIds": participantIDs, "lastMessageId": item.LastMessageID,
		"lastMessageText": item.LastMessageText, "lastMessageAt": item.LastMessageAt,
		"lastSenderId": item.LastSenderID, "typingUserId": item.TypingUserID,
		"typingExpiresAt": item.TypingExpiresAt, "activeParticipantIds": item.ActiveParticipantIDs,
		"lastReadAtCustomer": item.LastReadAtCustomer, "lastReadAtBusiness": item.LastReadAtBusiness,
		"unreadCustomerCount": item.UnreadCustomerCount, "unreadBusinessCount": item.UnreadBusinessCount,
		"createdAt": item.CreatedAt, "updatedAt": item.UpdatedAt,
	}
}

func messageResponse(item domain.Message) map[string]any {
	return map[string]any{"id": item.ID, "conversationId": item.ConversationID, "senderId": item.SenderID, "text": item.Text, "createdAt": item.CreatedAt}
}

func conversationPageResponse(page application.ConversationPage) map[string]any {
	items := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, conversationResponse(item))
	}
	return map[string]any{"items": items, "nextCursor": encodeCursor(page.NextCursor)}
}

func messagePageResponse(page application.MessagePage) map[string]any {
	items := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, messageResponse(item))
	}
	return map[string]any{"items": items, "nextCursor": encodeCursor(page.NextCursor)}
}
