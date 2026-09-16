package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/chat/domain"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	usersdomain "github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
)

type commandStub struct {
	conversation domain.Conversation
	message      domain.Message
	customerID   string
	text         string
}

func (s *commandStub) GetOrCreate(_ context.Context, _ application.Actor, _ uuid.UUID, customerID string) (domain.Conversation, error) {
	s.customerID = customerID
	return s.conversation, nil
}
func (s *commandStub) SendMessage(_ context.Context, _ string, _, _ uuid.UUID, text string) (domain.Message, error) {
	s.text = text
	return s.message, nil
}
func (s *commandStub) MarkRead(context.Context, string, uuid.UUID) error { return nil }
func (s *commandStub) SetTyping(context.Context, string, uuid.UUID, *time.Time) error {
	return nil
}
func (s *commandStub) SetActive(context.Context, string, uuid.UUID, *time.Time) error {
	return nil
}

type queryStub struct{ conversation domain.Conversation }

func (s *queryStub) Get(context.Context, string, uuid.UUID) (domain.Conversation, error) {
	return s.conversation, nil
}
func (s *queryStub) List(context.Context, string, *application.Cursor, int) ([]domain.Conversation, error) {
	return []domain.Conversation{s.conversation}, nil
}
func (s *queryStub) ListMessages(context.Context, string, uuid.UUID, *application.Cursor, int) ([]domain.Message, error) {
	return []domain.Message{}, nil
}
func (s *queryStub) UnreadCount(context.Context, string) (int, error) { return 2, nil }

type updatesStub struct{ updates chan application.Change }

func (s *updatesStub) SubscribeChanges(string) (<-chan application.Change, func()) {
	return s.updates, func() {}
}

func TestChatHTTPContract(t *testing.T) {
	conversationID := uuid.New()
	businessID := uuid.New()
	messageID := uuid.New()
	now := time.Now().UTC()
	conversation := domain.Conversation{
		ID: conversationID, BusinessID: &businessID, BusinessOwnerID: "provider-1", CustomerID: "customer-1",
		BusinessName: "Business", CustomerName: "Customer", CreatedAt: now, UpdatedAt: now,
	}
	commands := &commandStub{
		conversation: conversation,
		message:      domain.Message{ID: messageID, ConversationID: conversationID, SenderID: "customer-1", Text: "Hello", CreatedAt: now},
	}
	handler := NewHandler(application.NewService(commands, &queryStub{conversation: conversation}, nil))
	role := usersdomain.UserRoleCustomer
	user := usersdomain.User{ID: "customer-1", Role: &role}

	t.Run("get or create derives authenticated customer", func(t *testing.T) {
		body := []byte(`{"businessId":"` + businessID.String() + `","customerId":""}`)
		request := usershttp.WithCurrentUser(httptest.NewRequest(http.MethodPost, "/v1/conversations", bytes.NewReader(body)), user)
		recorder := httptest.NewRecorder()
		handler.GetOrCreate(recorder, request)
		if recorder.Code != http.StatusOK || commands.customerID != user.ID {
			t.Fatalf("status = %d, customer = %q, body = %s", recorder.Code, commands.customerID, recorder.Body.String())
		}
		if responseObject(t, recorder)["id"] != conversationID.String() {
			t.Fatalf("response = %s", recorder.Body.String())
		}
	})

	t.Run("send accepts client message id", func(t *testing.T) {
		body := []byte(`{"id":"` + messageID.String() + `","text":"  Hello  "}`)
		request := conversationRequestForTest(http.MethodPost, conversationID, body, user)
		recorder := httptest.NewRecorder()
		handler.SendMessage(recorder, request)
		if recorder.Code != http.StatusCreated || commands.text != "Hello" {
			t.Fatalf("status = %d, text = %q, body = %s", recorder.Code, commands.text, recorder.Body.String())
		}
	})

	t.Run("unread count", func(t *testing.T) {
		request := usershttp.WithCurrentUser(httptest.NewRequest(http.MethodGet, "/v1/conversations/unread-count", nil), user)
		recorder := httptest.NewRecorder()
		handler.UnreadCount(recorder, request)
		if recorder.Code != http.StatusOK || responseObject(t, recorder)["count"] != float64(2) {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestChatStreamStartsWithAuthoritativeSync(t *testing.T) {
	role := usersdomain.UserRoleCustomer
	user := usersdomain.User{ID: "customer-1", Role: &role}
	updates := &updatesStub{updates: make(chan application.Change, 1)}
	handler := NewHandler(application.NewService(&commandStub{}, &queryStub{}, updates))
	request := usershttp.WithCurrentUser(httptest.NewRequest(http.MethodGet, "/v1/chat/stream", nil), user)
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	recorder := httptest.NewRecorder()

	handler.Stream(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("status = %d, headers = %#v", recorder.Code, recorder.Header())
	}
	if recorder.Body.String() != "event: chat_sync\ndata: {}\n\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func conversationRequestForTest(method string, id uuid.UUID, body []byte, user usersdomain.User) *http.Request {
	request := usershttp.WithCurrentUser(httptest.NewRequest(method, "/v1/conversations/"+id.String(), bytes.NewReader(body)), user)
	route := chi.NewRouteContext()
	route.URLParams.Add("conversationID", id.String())
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}

func responseObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}
