package customerprofile

import (
	"context"
	"errors"
	"testing"

	"noodu-login-service/internal/customers"
)

type testRepository struct {
	exists  bool
	profile customers.Profile
	err     error
	created bool
}

type testUploader struct {
	calls int
}

func (uploader *testUploader) Upload(_ context.Context, image ImageUpload) (UploadedImage, error) {
	uploader.calls++
	return UploadedImage{
		ID:  image.Filename,
		URL: "https://i.ibb.co/" + image.Filename + "/image.jpg",
	}, nil
}

func (repository *testRepository) ProfileExistsByUserID(context.Context, int64) (bool, error) {
	return repository.exists, repository.err
}

func (repository *testRepository) CreateProfile(_ context.Context, profile customers.Profile) (customers.Profile, error) {
	repository.created = true
	repository.profile = profile
	if repository.err != nil {
		return customers.Profile{}, repository.err
	}
	profile.ID = 12
	return profile, nil
}

func (repository *testRepository) ProfileByUserID(context.Context, int64) (customers.Profile, error) {
	return repository.profile, repository.err
}

func testDocuments() Documents {
	image := func(name string) ImageUpload {
		return ImageUpload{Filename: name, ContentType: "image/jpeg", Data: []byte("valid image")}
	}
	return Documents{KTP: image("ktp"), Selfie: image("selfie"), Payslip: image("payslip")}
}

func TestCreateNormalizesAndPersistsProfile(t *testing.T) {
	repository := &testRepository{}
	uploader := &testUploader{}
	service := NewService(repository, uploader)
	address := "  Jalan Merdeka  "
	birthDate := "1995-02-01"
	income := 12500000.0
	created, err := service.Create(context.Background(), 7, CreateInput{
		NIK:           " 1234567890123456 ",
		Address:       &address,
		DateOfBirth:   &birthDate,
		MonthlyIncome: &income,
	}, testDocuments())
	if err != nil {
		t.Fatal(err)
	}
	if !repository.created || repository.profile.UserID != 7 {
		t.Fatalf("profile was not persisted with authenticated user id: %+v", repository.profile)
	}
	if repository.profile.NIK != "1234567890123456" {
		t.Errorf("NIK = %q, want trimmed value", repository.profile.NIK)
	}
	if repository.profile.Address == nil || *repository.profile.Address != "Jalan Merdeka" {
		t.Errorf("address = %v, want trimmed address", repository.profile.Address)
	}
	if created.ID != 12 || created.UserID != 7 {
		t.Errorf("created profile = %+v, want ID 12 and user ID 7", created)
	}
	if uploader.calls != 3 {
		t.Errorf("image upload calls = %d, want 3", uploader.calls)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	repository := &testRepository{}
	service := NewService(repository, &testUploader{})
	badDate := "01-02-1995"
	negativeIncome := -1.0
	tests := []struct {
		name  string
		user  int64
		input CreateInput
	}{
		{name: "invalid user id", user: 0, input: CreateInput{NIK: "1234567890123456"}},
		{name: "invalid NIK", user: 7, input: CreateInput{NIK: "123"}},
		{name: "invalid date", user: 7, input: CreateInput{NIK: "1234567890123456", DateOfBirth: &badDate}},
		{name: "negative income", user: 7, input: CreateInput{NIK: "1234567890123456", MonthlyIncome: &negativeIncome}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), test.user, test.input, testDocuments())
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
	if repository.created {
		t.Fatal("invalid input must not be persisted")
	}
}

func TestCreateRejectsExistingProfile(t *testing.T) {
	repository := &testRepository{exists: true}
	uploader := &testUploader{}
	service := NewService(repository, uploader)
	_, err := service.Create(context.Background(), 7, CreateInput{NIK: "1234567890123456"}, testDocuments())
	if !errors.Is(err, ErrProfileExists) {
		t.Fatalf("error = %v, want ErrProfileExists", err)
	}
	if repository.created {
		t.Fatal("existing profile must not be persisted again")
	}
	if uploader.calls != 0 {
		t.Fatal("existing profile must not trigger image uploads")
	}
}

func TestCreateValidatesAllImagesBeforeUploading(t *testing.T) {
	repository := &testRepository{}
	uploader := &testUploader{}
	service := NewService(repository, uploader)
	documents := testDocuments()
	documents.Payslip.ContentType = "text/plain"

	_, err := service.Create(context.Background(), 7, CreateInput{NIK: "1234567890123456"}, documents)
	if !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("error = %v, want ErrInvalidImage", err)
	}
	if uploader.calls != 0 {
		t.Fatalf("upload calls = %d, want 0 for invalid document set", uploader.calls)
	}
	if repository.created {
		t.Fatal("invalid document set must not be persisted")
	}
}
