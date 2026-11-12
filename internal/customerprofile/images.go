package customerprofile

import (
	"context"
	"errors"
	"fmt"

	"noodu-login-service/internal/customers"
)

var ErrInvalidImage = errors.New("gambar profil tidak valid")

type ImageUpload struct {
	Filename    string
	ContentType string
	Data        []byte
}

type ImageUploader interface {
	Upload(context.Context, ImageUpload) (UploadedImage, error)
}

type UploadedImage struct {
	ID  string
	URL string
}

type Documents struct {
	KTP     ImageUpload
	Selfie  ImageUpload
	Payslip ImageUpload
}

func (service *Service) uploadDocuments(ctx context.Context, profile customers.Profile, documents Documents) (customers.Profile, error) {
	uploads := []struct {
		input ImageUpload
		set   func(UploadedImage)
	}{
		{documents.KTP, func(image UploadedImage) {
			profile.KTPImageID = stringPointer(image.ID)
			profile.KTPImageURL = stringPointer(image.URL)
		}},
		{documents.Selfie, func(image UploadedImage) {
			profile.SelfieImageID = stringPointer(image.ID)
			profile.SelfieImageURL = stringPointer(image.URL)
		}},
		{documents.Payslip, func(image UploadedImage) {
			profile.PayslipImageID = stringPointer(image.ID)
			profile.PayslipImageURL = stringPointer(image.URL)
		}},
	}
	for _, upload := range uploads {
		if len(upload.input.Data) == 0 || !validImageContentType(upload.input.ContentType) {
			return customers.Profile{}, ErrInvalidImage
		}
	}
	for _, upload := range uploads {
		image, err := service.uploader.Upload(ctx, upload.input)
		if err != nil {
			return customers.Profile{}, fmt.Errorf("upload profile image: %w", err)
		}
		upload.set(image)
	}
	return profile, nil
}

func stringPointer(value string) *string {
	return &value
}

func validImageContentType(value string) bool {
	switch value {
	case "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}
