package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"noodu-login-service/internal/auth"
	"noodu-login-service/internal/customerprofile"
	"noodu-login-service/internal/users"

	"golang.org/x/crypto/bcrypt"
)

type userStore interface {
	ByEmail(context.Context, string) (users.User, error)
	ByID(context.Context, int64) (users.User, error)
	Create(context.Context, string, string, string) (users.User, error)
	UpdateProfile(context.Context, int64, *string, *string) (users.User, error)
	Delete(context.Context, int64) error
}

type pinger interface {
	PingContext(context.Context) error
}

type Server struct {
	store            userStore
	db               pinger
	tokens           *auth.TokenManager
	origin           string
	limiter          *loginLimiter
	customerProfiles *customerprofile.Service
}

func New(store userStore, db pinger, tokens *auth.TokenManager, origin string) *Server {
	return &Server{store: store, db: db, tokens: tokens, origin: origin, limiter: newLoginLimiter(10, time.Minute)}
}

func NewWithCustomerProfiles(store userStore, db pinger, tokens *auth.TokenManager, origin string, customerProfiles *customerprofile.Service) *Server {
	server := New(store, db, tokens, origin)
	server.customerProfiles = customerProfiles
	return server
}

func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /readyz", server.ready)
	mux.HandleFunc("POST /api/v1/auth/register", server.register)
	mux.HandleFunc("POST /api/v1/auth/login", server.login)
	mux.HandleFunc("DELETE /api/v1/auth/account", server.deleteAccount)
	mux.Handle("GET /api/v1/auth/me", server.requireAuth(http.HandlerFunc(server.me)))
	mux.Handle("PATCH /api/v1/auth/me", server.requireAuth(http.HandlerFunc(server.updateMe)))
	mux.Handle("DELETE /api/v1/auth/me", server.requireAuth(http.HandlerFunc(server.deleteMe)))
	if server.customerProfiles != nil {
		mux.Handle("POST /api/v1/customers/profile", server.requireAuth(http.HandlerFunc(server.createCustomerProfile)))
		mux.Handle("GET /api/v1/customers/profile", server.requireAuth(http.HandlerFunc(server.getCustomerProfile)))
	}
	return server.securityHeaders(server.cors(mux))
}

func (server *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) ready(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if err := server.db.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "Database belum siap")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (server *Server) login(w http.ResponseWriter, request *http.Request) {
	if !server.limiter.Allow(clientIP(request)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan login. Coba lagi dalam satu menit.")
		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, request, &input); err != nil || !validEmail(input.Email) || len(input.Password) == 0 || len(input.Password) > 72 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Email dan password harus diisi dengan format yang valid")
		return
	}

	user, err := server.store.ByEmail(request.Context(), input.Email)
	if err != nil || !user.IsActive || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")
		return
	}

	accessToken, expiresAt, err := server.tokens.Issue(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error", "Token tidak dapat dibuat")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_at":   expiresAt,
		"user":         publicUser(user),
	}})
}

func (server *Server) register(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, request, &input); err != nil || !validUsername(input.Username) || !validEmail(input.Email) || len(input.Password) < 12 || len(input.Password) > 72 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Username harus 3-50 karakter, email valid, dan password 12-72 byte")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password_error", "Akun tidak dapat dibuat")
		return
	}
	user, err := server.store.Create(request.Context(), input.Username, input.Email, string(passwordHash))
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "account_exists", "Username atau email sudah digunakan")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Akun tidak dapat dibuat")
		return
	}
	accessToken, expiresAt, err := server.tokens.Issue(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error", "Token tidak dapat dibuat")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_at":   expiresAt,
		"user":         publicUser(user),
	}})
}

func (server *Server) me(w http.ResponseWriter, request *http.Request) {
	claims := request.Context().Value(claimsContextKey{}).(*auth.Claims)
	userID, _ := strconv.ParseInt(claims.Subject, 10, 64)
	user, err := server.store.ByID(request.Context(), userID)
	if errors.Is(err, users.ErrNotFound) || (err == nil && !user.IsActive) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak berlaku")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Permintaan tidak dapat diproses")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": publicUser(user)})
}

func (server *Server) updateMe(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Username *string `json:"username"`
		Email    *string `json:"email"`
	}
	if err := decodeJSON(w, request, &input); err != nil || (input.Username == nil && input.Email == nil) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Isi minimal username atau email yang ingin diubah")
		return
	}
	if input.Username != nil && !validUsername(*input.Username) {
		writeError(w, http.StatusBadRequest, "invalid_username", "Username harus 3-50 karakter dan hanya boleh berisi huruf, angka, titik, garis bawah, atau tanda hubung")
		return
	}
	if input.Email != nil && !validEmail(*input.Email) {
		writeError(w, http.StatusBadRequest, "invalid_email", "Format email tidak valid")
		return
	}

	userID := authenticatedUserID(request)
	user, err := server.store.ByID(request.Context(), userID)
	if errors.Is(err, users.ErrNotFound) || (err == nil && !user.IsActive) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak berlaku")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Permintaan tidak dapat diproses")
		return
	}

	updated, err := server.store.UpdateProfile(request.Context(), userID, input.Username, input.Email)
	if errors.Is(err, users.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak berlaku")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "account_exists", "Username atau email sudah digunakan")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Profil tidak dapat diperbarui")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": publicUser(updated)})
}

func (server *Server) deleteMe(w http.ResponseWriter, request *http.Request) {
	userID := authenticatedUserID(request)
	user, err := server.store.ByID(request.Context(), userID)
	if errors.Is(err, users.ErrNotFound) || (err == nil && !user.IsActive) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak berlaku")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Permintaan tidak dapat diproses")
		return
	}
	if err := server.store.Delete(request.Context(), userID); errors.Is(err, users.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak berlaku")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Akun tidak dapat dihapus")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) deleteAccount(w http.ResponseWriter, request *http.Request) {
	if !server.limiter.Allow(clientIP(request)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan. Coba lagi dalam satu menit.")
		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, request, &input); err != nil || !validEmail(input.Email) || len(input.Password) == 0 || len(input.Password) > 72 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Email dan password harus diisi dengan format yang valid")
		return
	}

	user, err := server.store.ByEmail(request.Context(), input.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")
		return
	}
	if err := server.store.Delete(request.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Akun tidak dapat dihapus")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		parts := strings.Fields(request.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Bearer token wajib disertakan")
			return
		}
		claims, err := server.tokens.Verify(parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak berlaku atau sudah kedaluwarsa")
			return
		}
		ctx := context.WithValue(request.Context(), claimsContextKey{}, claims)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

func (server *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			if server.origin == "" || origin != server.origin {
				writeError(w, http.StatusForbidden, "origin_not_allowed", "Origin tidak diizinkan")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", server.origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, request)
	})
}

func (server *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, request)
	})
}

type claimsContextKey struct{}

func decodeJSON(w http.ResponseWriter, request *http.Request, destination any) error {
	if !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
		return errors.New("content type harus application/json")
	}
	request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("hanya satu objek JSON yang diperbolehkan")
	}
	return nil
}

func validEmail(value string) bool {
	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	return len(value) <= 100 && err == nil && parsed.Address == strings.TrimSpace(value) && strings.Contains(parsed.Address, "@")
}

func validUsername(value string) bool {
	if len(value) < 3 || len(value) > 50 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func authenticatedUserID(request *http.Request) int64 {
	claims := request.Context().Value(claimsContextKey{}).(*auth.Claims)
	userID, _ := strconv.ParseInt(claims.Subject, 10, 64)
	return userID
}

func isUniqueViolation(err error) bool {
	var stateError interface{ SQLState() string }
	return errors.As(err, &stateError) && stateError.SQLState() == "23505"
}

func publicUser(user users.User) map[string]any {
	return map[string]any{"id": user.ID, "username": user.Username, "email": user.Email}
}

func clientIP(request *http.Request) string {
	address, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return address
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
