package postgres

import (
	"time"

	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
)

const conversationColumns = `c.id,c.business_id,c.legacy_business_id,c.business_owner_id,c.customer_id,
	c.business_name_snapshot,c.business_image_path_snapshot,c.customer_name_snapshot,c.customer_image_path_snapshot,
	c.last_message_id,c.last_message_text,c.last_message_at,m.sender_id,
	cs.last_read_at,bs.last_read_at,cs.unread_count,bs.unread_count,
	cs.typing_until,bs.typing_until,cs.active_until,bs.active_until,c.created_at,c.updated_at`

const conversationJoins = `
	FROM chat_conversations c
	JOIN chat_participant_state cs ON cs.conversation_id=c.id AND cs.participant_role='customer'
	JOIN chat_participant_state bs ON bs.conversation_id=c.id AND bs.participant_role='business'
	LEFT JOIN chat_messages m ON m.id=c.last_message_id`

type scanner interface{ Scan(...any) error }

func scanConversation(row scanner) (domain.Conversation, error) {
	var item domain.Conversation
	var customerTyping, businessTyping, customerActive, businessActive *time.Time
	err := row.Scan(
		&item.ID, &item.BusinessID, &item.LegacyBusinessID, &item.BusinessOwnerID, &item.CustomerID,
		&item.BusinessName, &item.BusinessImagePath, &item.CustomerName, &item.CustomerImagePath,
		&item.LastMessageID, &item.LastMessageText, &item.LastMessageAt, &item.LastSenderID,
		&item.LastReadAtCustomer, &item.LastReadAtBusiness,
		&item.UnreadCustomerCount, &item.UnreadBusinessCount,
		&customerTyping, &businessTyping, &customerActive, &businessActive,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return domain.Conversation{}, err
	}

	now := time.Now()
	if customerTyping != nil && customerTyping.After(now) {
		item.TypingUserID = &item.CustomerID
		item.TypingExpiresAt = customerTyping
	} else if businessTyping != nil && businessTyping.After(now) {
		item.TypingUserID = &item.BusinessOwnerID
		item.TypingExpiresAt = businessTyping
	}
	item.ActiveParticipantIDs = make([]string, 0, 2)
	if customerActive != nil && customerActive.After(now) {
		item.ActiveParticipantIDs = append(item.ActiveParticipantIDs, item.CustomerID)
	}
	if businessActive != nil && businessActive.After(now) {
		item.ActiveParticipantIDs = append(item.ActiveParticipantIDs, item.BusinessOwnerID)
	}
	return item, nil
}
