package httpapi

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"noodu-login-service/internal/auth"
	"noodu-login-service/internal/customerprofile"
	"noodu-login-service/internal/customers"
	"noodu-login-service/internal/users"
)

type customerProfileRepository struct {
	profile customers.Profile
}

type customerProfileUploader struct{}

func (customerProfileUploader) Upload(_ context.Context, image customerprofile.ImageUpload) (customerprofile.UploadedImage, error) {
	return customerprofile.UploadedImage{
		ID:  image.Filename,
		URL: "https://i.ibb.co/" + image.Filename + "/image.jpg",
	}, nil
}

func (repository *customerProfileRepository) ProfileExistsByUserID(context.Context, int64) (bool, error) {
	return false, nil
}

func (repository *customerProfileRepository) CreateProfile(_ context.Context, profile customers.Profile) (customers.Profile, error) {
	repository.profile = profile
	profile.ID = 14
	return profile, nil
}

func (repository *customerProfileRepository) ProfileByUserID(_ context.Context, userID int64) (customers.Profile, error) {
	profile := repository.profile
	profile.UserID = userID
	return profile, nil
}

func TestCreateCustomerProfileUsesAuthenticatedUserID(t *testing.T) {
	userStore := &testStore{user: users.User{ID: 7, Username: "alice", Email: "alice@example.com", IsActive: true}}
	repository := &customerProfileRepository{}
	tokens := auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute)
	server := NewWithCustomerProfiles(userStore, testPinger{}, tokens, "", customerprofile.NewService(repository, customerProfileUploader{}))
	token, _, err := tokens.Issue(userStore.user.ID)
	if err != nil {
		t.Fatal(err)
	}

	request := newMultipartProfileRequest(t, true)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if repository.profile.UserID != userStore.user.ID {
		t.Fatalf("persisted user ID = %d, want token subject %d", repository.profile.UserID, userStore.user.ID)
	}
	if repository.profile.KTPImageID == nil || repository.profile.KTPImageURL == nil {
		t.Fatal("response profile should include image ID and URL")
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/customers/profile", nil)
	getRequest.Header.Set("Authorization", "Bearer "+token)
	getResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body: %s", getResponse.Code, http.StatusOK, getResponse.Body.String())
	}
	if !strings.Contains(getResponse.Body.String(), "ktp_image_url") {
		t.Fatal("GET response should include image URL for the frontend")
	}
}

func newMultipartProfileRequest(t *testing.T, includeUserID bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	var imageBytes bytes.Buffer
	testImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	testImage.Set(0, 0, color.White)
	if err := png.Encode(&imageBytes, testImage); err != nil {
		t.Fatal(err)
	}
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"nik":           "1234567890123456",
		"address":       "Jalan Merdeka",
		"ktp_image":     "ktp.jpg",
		"selfie_image":  "selfie.jpg",
		"payslip_image": "payslip.jpg",
	}
	if includeUserID {
		fields["user_id"] = "999"
	}
	for field, filename := range fields {
		if field == "ktp_image" || field == "selfie_image" || field == "payslip_image" {
			part, err := writer.CreateFormFile(field, filename)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(imageBytes.Bytes()); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := writer.WriteField(field, filename); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/customers/profile", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
