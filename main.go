package main

import (
	"database/sql"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/billbatista/acasinha-expenses/eventlogger"
	"github.com/billbatista/acasinha-expenses/middleware"
	"github.com/billbatista/acasinha-expenses/session"
	"github.com/billbatista/acasinha-expenses/user"
	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=expenses sslmode=disable")
	if err != nil {
		printErrorAndExit("database connection", err)
	}
	err = db.Ping()
	if err != nil {
		printErrorAndExit("pinging database", err)
	}

	evtlogger := eventlogger.NewSqlEventLogger(db)
	worker := eventlogger.NewWorker(evtlogger, 100)
	worker.Start()
	defer worker.Shutdown()

	userRepo := user.NewRepository(db)
	sessionRepo := session.NewRepository(db)

	router := chi.NewRouter()
	router.Use(chimiddleware.Logger)
	router.Use(middleware.AuthMiddleware(sessionRepo)) // Add auth middleware globally

	workDir, _ := os.Getwd()
	staticDir := http.Dir(filepath.Join(workDir, "./static"))
	router.Get("/favicon.ico", http.FileServer(staticDir).ServeHTTP)

	// Public routes
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if middleware.IsAuthenticated(r.Context()) {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		tmpl, err := template.ParseFiles("templates/base.html", "templates/index.html")
		if err != nil {
			slog.Error("failed to parse template", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		tmpl.ExecuteTemplate(w, "base.html", nil)
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		evt := eventlogger.NewEvent(
			eventlogger.WithType("health_request"),
			eventlogger.WithData(map[string]string{
				"message":     "ok",
				"http_status": strconv.Itoa(http.StatusOK),
			}),
		)
		worker.Log(evt)
		w.Write([]byte("ok"))
	})

	router.Post("/user/login", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		userdb, err := userRepo.GetByEmail(ctx, email)
		if err != nil {
			slog.Error("failed to fetch user", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if userdb == nil {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		validPassword := userRepo.VerifyPassword(userdb.PasswordHash, password)
		if validPassword != nil {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		sess, err := sessionRepo.Create(ctx, userdb.ID)
		if err != nil {
			slog.Error("failed to create session", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     session.CookieName,
			Value:    sess.Token,
			Path:     "/",
			Expires:  sess.ExpiresAt,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		evt := eventlogger.NewEvent(
			eventlogger.WithType("user.logged_in"),
			eventlogger.WithData(map[string]string{
				"user_id":    userdb.ID.String(),
				"email":      userdb.Email,
				"session_id": sess.ID.String(),
			}),
		)
		worker.Log(evt)

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})
	router.Post("/user/register", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		registeredUser, err := userRepo.Register(ctx, email, password)
		if err != nil {
			switch err {
			case user.ErrEmailExists:
				http.Error(w, err.Error(), http.StatusConflict)
			case user.ErrBlankPassword, user.ErrInvalidEmail:
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				slog.Error("failed to register user", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		sess, err := sessionRepo.Create(ctx, registeredUser.ID)
		if err != nil {
			slog.Error("failed to create session", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     session.CookieName,
			Value:    sess.Token,
			Path:     "/",
			Expires:  sess.ExpiresAt,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		evt := eventlogger.NewEvent(
			eventlogger.WithType("user.registered"),
			eventlogger.WithData(map[string]string{
				"user_id":    registeredUser.ID.String(),
				"email":      registeredUser.Email,
				"session_id": sess.ID.String(),
			}),
		)
		worker.Log(evt)

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	// Protected routes - require authentication
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth("/"))

		r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			userID, _ := middleware.GetUserID(r.Context())
			w.Write([]byte("Welcome to dashboard! User ID: " + userID.String()))
		})

		r.Get("/user/profile", func(w http.ResponseWriter, r *http.Request) {
			userID, _ := middleware.GetUserID(r.Context())

			user, err := userRepo.GetByID(r.Context(), userID)
			if err != nil {
				slog.Error("failed to fetch user", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			tmpl, err := template.ParseFiles("templates/base.html", "templates/profile.html")
			if err != nil {
				slog.Error("failed to parse template", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			data := map[string]any{
				"User": user,
			}

			tmpl.ExecuteTemplate(w, "base.html", data)
		})

		r.Get("/user/profile/avatar", func(w http.ResponseWriter, r *http.Request) {
			userID, _ := middleware.GetUserID(r.Context())

			user, err := userRepo.GetByID(r.Context(), userID)
			if err != nil {
				slog.Error("failed to fetch user", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("content-type", "image/jpeg")
			w.Write(user.Avatar)
		})

		r.Post("/user/profile/update-name", func(w http.ResponseWriter, r *http.Request) {
			userID, _ := middleware.GetUserID(r.Context())

			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}

			name := r.FormValue("name")

			err := userRepo.UpdateName(r.Context(), userID, name)
			if err != nil {
				slog.Error("failed to update name", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			evt := eventlogger.NewEvent(
				eventlogger.WithType("user.name_updated"),
				eventlogger.WithData(map[string]string{
					"user_id": userID.String(),
					"name":    name,
				}),
			)
			worker.Log(evt)

			http.Redirect(w, r, "/user/profile?success=Name updated successfully", http.StatusSeeOther)
		})

		r.Post("/user/profile/update-avatar", func(w http.ResponseWriter, r *http.Request) {
			userID, _ := middleware.GetUserID(r.Context())

			// sets the max memory limit to 10 MB (10 shifted left by 20 bits = 10 * 1,048,576 bytes)
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}
			file, handler, err := r.FormFile("avatar")
			if err != nil {
				slog.Error("retrieving form file", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			imgBytes, err := io.ReadAll(file)
			if err != nil {
				slog.Error("reading file", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			err = userRepo.UpdateAvatar(r.Context(), imgBytes, userID)
			if err != nil {
				slog.Error("failed to update avatar", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			evt := eventlogger.NewEvent(
				eventlogger.WithType("user.avatar_updated"),
				eventlogger.WithData(map[string]string{
					"user_id": userID.String(),
					"file":    handler.Filename,
				}),
			)
			worker.Log(evt)

			http.Redirect(w, r, "/user/profile?success=Avatar updated successfully", http.StatusSeeOther)
		})

		r.Post("/user/logout", func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.CookieName)
			if err == nil {
				sessionRepo.Delete(r.Context(), cookie.Value)
			}

			http.SetCookie(w, &http.Cookie{
				Name:   session.CookieName,
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})

			http.Redirect(w, r, "/", http.StatusSeeOther)
		})
	})

	slog.Info("server starting", "port", 5000)
	http.ListenAndServe(":5000", router)
}

func printErrorAndExit(msg string, e error) {
	slog.Error(msg, "error", e)
	os.Exit(1)
}
