package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type contextKey string

const tokenContextKey contextKey = "firebase-token"

type FirebaseAuthenticator struct {
	client *auth.Client
	app    *firebase.App
	logger *slog.Logger
}

func NewFirebaseAuthenticator(ctx context.Context, projectID string, logger *slog.Logger) (*FirebaseAuthenticator, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase auth client: %w", err)
	}

	return &FirebaseAuthenticator{client: client, app: app, logger: logger}, nil
}

func (a *FirebaseAuthenticator) MessagingClient(ctx context.Context) (*messaging.Client, error) {
	return a.app.Messaging(ctx)
}

func (a *FirebaseAuthenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			httpx.WriteRequestError(w, r, http.StatusUnauthorized, "unauthenticated", "missing bearer token")
			return
		}

		token, err := a.client.VerifyIDTokenAndCheckRevoked(r.Context(), rawToken)
		if err != nil {
			a.logger.Warn("Firebase ID token verification failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
			httpx.WriteRequestError(w, r, http.StatusUnauthorized, "unauthenticated", "invalid or revoked Firebase ID token")
			return
		}

		ctx := context.WithValue(r.Context(), tokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentToken(ctx context.Context) (*auth.Token, bool) {
	token, ok := ctx.Value(tokenContextKey).(*auth.Token)
	return token, ok
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
