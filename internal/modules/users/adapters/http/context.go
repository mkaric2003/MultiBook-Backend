package http

import (
	"context"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/users/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/auth"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
)

type userContextKey struct{}

func ProvisionUser(service *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := auth.CurrentToken(r.Context())
			if !ok {
				httpx.WriteRequestError(w, r, http.StatusInternalServerError, "authenticated_user_unavailable", "authenticated user is unavailable")
				return
			}
			user, err := service.CurrentUser(r.Context(), application.AuthenticatedUser{
				ID:          token.UID,
				Email:       tokenClaim(token, "email"),
				DisplayName: tokenClaim(token, "name"),
			})
			if err != nil {
				httpx.WriteRequestError(w, r, http.StatusInternalServerError, "user_unavailable", "could not load the current user")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
		})
	}
}

func CurrentUser(r *http.Request) (domain.User, bool) {
	user, ok := r.Context().Value(userContextKey{}).(domain.User)
	return user, ok
}

// RequireCurrentUser returns the provisioned user or writes the standard
// internal error used when authenticated middleware context is unavailable.
func RequireCurrentUser(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	user, ok := CurrentUser(r)
	if !ok {
		httpx.WriteRequestError(w, r, http.StatusInternalServerError, "authenticated_user_unavailable", "authenticated user is unavailable")
		return domain.User{}, false
	}
	return user, true
}

// WithCurrentUser attaches a provisioned user to a request. It is primarily
// useful for HTTP-adapter tests that exercise handlers after authentication
// and provisioning have completed.
func WithCurrentUser(r *http.Request, user domain.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey{}, user))
}

func RequireRoles(roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r)
			if !ok || user.Role == nil {
				httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "a marketplace role is required")
				return
			}
			if _, ok := allowed[*user.Role]; !ok {
				httpx.WriteRequestError(w, r, http.StatusForbidden, "forbidden", "you are not authorized for this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func tokenClaim(token *firebaseauth.Token, key string) string {
	// Kept private to this HTTP adapter: the application layer does not depend
	// on Firebase's token type.
	value, _ := token.Claims[key].(string)
	return strings.TrimSpace(value)
}
