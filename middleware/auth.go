package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/billbatista/acasinha-expenses/session"
	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// AuthMiddleware checks if user has a valid session
func AuthMiddleware(sessionRepo session.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.CookieName)
			if err != nil {
				slog.Info("no cookie found, not authenticated")
				next.ServeHTTP(w, r)
				return
			}

			sess, err := sessionRepo.GetByToken(r.Context(), cookie.Value)
			if err != nil {
				slog.Info("invalid/expired session")
				http.SetCookie(w, &http.Cookie{
					Name:   session.CookieName,
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				})
				next.ServeHTTP(w, r)
				return
			}

			// Valid session - add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, sess.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth redirects to login if user is not authenticated
func RequireAuth(redirectTo string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey)
			if userID == nil {
				http.Redirect(w, r, redirectTo, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

// IsAuthenticated checks if user is authenticated
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetUserID(ctx)
	return ok
}
