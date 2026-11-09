package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"noodu-login-service/internal/auth"
	"noodu-login-service/internal/users"

	"golang.org/x/crypto/bcrypt"
)

type testStore struct {
	user    users.User
	deleted bool
}

func (store *testStore) ByEmail(context.Context, string) (users.User, error) {
	return store.user, nil
}

func (store *testStore) ByID(context.Context, int64) (users.User, error) {
	return store.user, nil
}

func (store *testStore) Create(_ context.Context, username, email, passwordHash string) (users.User, error) {
	store.user = users.User{ID: 8, Username: username, Email: email, PasswordHash: passwordHash, IsActive: true}
	return store.user, nil
}

func (store *testStore) UpdateProfile(_ context.Context, _ int64, username, email *string) (users.User, error) {
	if username != nil {
		store.user.Username = *username
	}
	if email != nil {
		store.user.Email = *email
	}
	return store.user, nil
}

func (store *testStore) Delete(context.Context, int64) error {
	store.deleted = true
	return nil
}

type testPinger struct{}

func (testPinger) PingContext(context.Context) error { return nil }

func TestLoginThenGetProfile(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct horse battery"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store := &testStore{user: users.User{
		ID:           7,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: string(passwordHash),
		IsActive:     true,
	}}
	server := New(store, testPinger{}, auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute), "")

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"alice@example.com","password":"correct horse battery"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", loginResponse.Code, http.StatusOK, loginResponse.Body.String())
	}

	var response struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Data.AccessToken == "" {
		t.Fatal("login response did not contain an access token")
	}

	profileRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	profileRequest.Header.Set("Authorization", "Bearer "+response.Data.AccessToken)
	profileResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(profileResponse, profileRequest)
	if profileResponse.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want %d; body: %s", profileResponse.Code, http.StatusOK, profileResponse.Body.String())
	}
}

func TestProfileRequiresBearerToken(t *testing.T) {
	server := New(&testStore{}, testPinger{}, auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute), "")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRegisterCreatesHashedUserAndToken(t *testing.T) {
	store := &testStore{}
	server := New(store, testPinger{}, auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute), "")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"username":"alice","email":"alice@example.com","password":"a-long-enough-password"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if store.user.PasswordHash == "a-long-enough-password" {
		t.Fatal("password was stored without hashing")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.user.PasswordHash), []byte("a-long-enough-password")); err != nil {
		t.Fatalf("stored password hash does not match input: %v", err)
	}
	var result struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Data.AccessToken == "" {
		t.Fatal("register response did not contain an access token")
	}
}

func TestProfileCanBeUpdatedAndHardDeleted(t *testing.T) {
	store := &testStore{user: users.User{ID: 7, Username: "alice", Email: "alice@example.com", IsActive: true}}
	server := New(store, testPinger{}, auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute), "")
	token, _, err := auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute).Issue(store.user.ID)
	if err != nil {
		t.Fatal(err)
	}

	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", strings.NewReader(`{"username":"alice_new"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}
	if store.user.Username != "alice_new" {
		t.Fatalf("username = %q, want %q", store.user.Username, "alice_new")
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/me", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
	if !store.deleted {
		t.Fatal("user should be permanently deleted")
	}
}

func TestSoftDeletedAccountCanBeHardDeletedWithoutBearer(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store := &testStore{user: users.User{
		ID:           9,
		Username:     "inactive",
		Email:        "inactive@example.com",
		PasswordHash: string(passwordHash),
		IsActive:     false,
	}}
	server := New(store, testPinger{}, auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute), "")
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/account", strings.NewReader(`{"email":"inactive@example.com","password":"correct-password"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if !store.deleted {
		t.Fatal("soft-deleted account should be permanently deleted")
	}
}

func TestHardDeleteWithoutBearerRejectsWrongPassword(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store := &testStore{user: users.User{ID: 9, Email: "inactive@example.com", PasswordHash: string(passwordHash)}}
	server := New(store, testPinger{}, auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute), "")
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/account", strings.NewReader(`{"email":"inactive@example.com","password":"wrong-password"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if store.deleted {
		t.Fatal("account must not be deleted with the wrong password")
	}
}

func TestEmailMustFitUsersTableColumn(t *testing.T) {
	if validEmail("alice@example.com") != true {
		t.Fatal("expected a valid email to be accepted")
	}
	if validEmail(strings.Repeat("a", 90) + "@example.com") {
		t.Fatal("expected email longer than 100 bytes to be rejected")
	}
}
