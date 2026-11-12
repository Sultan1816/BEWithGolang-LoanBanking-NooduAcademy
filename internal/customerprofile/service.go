package customerprofile

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"noodu-login-service/internal/customers"
)

var (
	ErrInvalidInput  = errors.New("data profil tidak valid")
	ErrProfileExists = errors.New("profil customer sudah terdaftar")
)

type Repository interface {
	ProfileExistsByUserID(context.Context, int64) (bool, error)
	CreateProfile(context.Context, customers.Profile) (customers.Profile, error)
	ProfileByUserID(context.Context, int64) (customers.Profile, error)
}

type CreateInput struct {
	NIK               string   `json:"nik"`
	Address           *string  `json:"address"`
	DateOfBirth       *string  `json:"date_of_birth"`
	MonthlyIncome     *float64 `json:"monthly_income"`
	Phone             *string  `json:"phone"`
	CurrentAddress    *string  `json:"current_address"`
	MotherMaidenName  *string  `json:"mother_maiden_name"`
	Occupation        *string  `json:"occupation"`
	PlaceOfBirth      *string  `json:"place_of_birth"`
	AccountNumber     *string  `json:"account_number"`
	AccountHolderName *string  `json:"account_holder_name"`
}

type Service struct {
	repository Repository
	uploader   ImageUploader
}

func NewService(repository Repository, uploader ImageUploader) *Service {
	return &Service{repository: repository, uploader: uploader}
}

func (service *Service) Create(ctx context.Context, userID int64, input CreateInput, documents Documents) (customers.Profile, error) {
	nik := strings.TrimSpace(input.NIK)
	if userID <= 0 || len(nik) != 16 || !allDigits(nik) {
		return customers.Profile{}, ErrInvalidInput
	}
	if input.MonthlyIncome != nil && (*input.MonthlyIncome < 0 || math.IsNaN(*input.MonthlyIncome) || math.IsInf(*input.MonthlyIncome, 0)) {
		return customers.Profile{}, ErrInvalidInput
	}

	var dateOfBirth *time.Time
	if input.DateOfBirth != nil {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*input.DateOfBirth))
		if err != nil || parsed.After(time.Now()) {
			return customers.Profile{}, ErrInvalidInput
		}
		dateOfBirth = &parsed
	}

	exists, err := service.repository.ProfileExistsByUserID(ctx, userID)
	if err != nil {
		return customers.Profile{}, fmt.Errorf("check customer profile: %w", err)
	}
	if exists {
		return customers.Profile{}, ErrProfileExists
	}
	if service.uploader == nil {
		return customers.Profile{}, errors.New("customer profile image uploader is not configured")
	}

	profile := customers.Profile{
		UserID:            userID,
		NIK:               nik,
		Address:           cleanOptional(input.Address),
		DateOfBirth:       dateOfBirth,
		MonthlyIncome:     input.MonthlyIncome,
		Phone:             cleanOptional(input.Phone),
		CurrentAddress:    cleanOptional(input.CurrentAddress),
		MotherMaidenName:  cleanOptional(input.MotherMaidenName),
		Occupation:        cleanOptional(input.Occupation),
		PlaceOfBirth:      cleanOptional(input.PlaceOfBirth),
		AccountNumber:     cleanOptional(input.AccountNumber),
		AccountHolderName: cleanOptional(input.AccountHolderName),
	}
	profile, err = service.uploadDocuments(ctx, profile, documents)
	if err != nil {
		return customers.Profile{}, err
	}
	created, err := service.repository.CreateProfile(ctx, profile)
	if err != nil {
		return customers.Profile{}, fmt.Errorf("create customer profile: %w", err)
	}
	return created, nil
}

func (service *Service) ByUserID(ctx context.Context, userID int64) (customers.Profile, error) {
	profile, err := service.repository.ProfileByUserID(ctx, userID)
	if err != nil {
		return customers.Profile{}, fmt.Errorf("get customer profile: %w", err)
	}
	return profile, nil
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func allDigits(value string) bool {
	for _, character := range value {
		if !unicode.IsDigit(character) || character > unicode.MaxASCII {
			return false
		}
	}
	return true
}
