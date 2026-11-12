package customers

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Profile struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	NIK               string     `json:"nik"`
	Address           *string    `json:"address"`
	DateOfBirth       *time.Time `json:"date_of_birth"`
	MonthlyIncome     *float64   `json:"monthly_income"`
	Phone             *string    `json:"phone"`
	CurrentAddress    *string    `json:"current_address"`
	MotherMaidenName  *string    `json:"mother_maiden_name"`
	Occupation        *string    `json:"occupation"`
	PlaceOfBirth      *string    `json:"place_of_birth"`
	KTPImageID        *string    `json:"ktp_image_id"`
	KTPImageURL       *string    `json:"ktp_image_url"`
	SelfieImageID     *string    `json:"selfie_image_id"`
	SelfieImageURL    *string    `json:"selfie_image_url"`
	PayslipImageID    *string    `json:"payslip_image_id"`
	PayslipImageURL   *string    `json:"payslip_image_url"`
	AccountNumber     *string    `json:"account_number"`
	AccountHolderName *string    `json:"account_holder_name"`
	CurrentTotalDebt  *float64   `json:"current_total_debt"`
	CreatedAt         *time.Time `json:"created_at"`
}

func (store *Store) ProfileExistsByUserID(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := store.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM customers WHERE user_id = $1)`,
		userID,
	).Scan(&exists)
	return exists, err
}

func (store *Store) CreateProfile(ctx context.Context, profile Profile) (Profile, error) {
	err := store.db.QueryRowContext(ctx,
		`INSERT INTO customers (
			user_id, nik, address, date_of_birth, monthly_income, phone,
			current_address, mother_maiden_name, occupation, place_of_birth,
			ktp_image_path, ktp_image_id, selfie_image_path, selfie_image_id,
			payslip_image_path, payslip_image_id,
			account_number, account_holder_name
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, user_id, nik, address, date_of_birth, monthly_income, phone,
		          current_address, mother_maiden_name, occupation, place_of_birth,
		          ktp_image_id, ktp_image_path, selfie_image_id, selfie_image_path,
		          payslip_image_id, payslip_image_path,
		          account_number, account_holder_name, current_total_debt, created_at`,
		profile.UserID, profile.NIK, profile.Address, profile.DateOfBirth,
		profile.MonthlyIncome, profile.Phone, profile.CurrentAddress,
		profile.MotherMaidenName, profile.Occupation, profile.PlaceOfBirth,
		profile.KTPImageURL, profile.KTPImageID, profile.SelfieImageURL,
		profile.SelfieImageID, profile.PayslipImageURL, profile.PayslipImageID,
		profile.AccountNumber, profile.AccountHolderName,
	).Scan(
		&profile.ID, &profile.UserID, &profile.NIK, &profile.Address,
		&profile.DateOfBirth, &profile.MonthlyIncome, &profile.Phone,
		&profile.CurrentAddress, &profile.MotherMaidenName, &profile.Occupation,
		&profile.PlaceOfBirth, &profile.KTPImageID, &profile.KTPImageURL,
		&profile.SelfieImageID, &profile.SelfieImageURL, &profile.PayslipImageID,
		&profile.PayslipImageURL, &profile.AccountNumber, &profile.AccountHolderName,
		&profile.CurrentTotalDebt, &profile.CreatedAt,
	)
	return profile, err
}

func (store *Store) ProfileByUserID(ctx context.Context, userID int64) (Profile, error) {
	var profile Profile
	err := store.db.QueryRowContext(ctx,
		`SELECT id, user_id, nik, address, date_of_birth, monthly_income, phone,
		        current_address, mother_maiden_name, occupation, place_of_birth,
		        ktp_image_id, ktp_image_path, selfie_image_id, selfie_image_path,
		        payslip_image_id, payslip_image_path, account_number, account_holder_name,
		        current_total_debt, created_at
		 FROM customers WHERE user_id = $1`,
		userID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.NIK, &profile.Address,
		&profile.DateOfBirth, &profile.MonthlyIncome, &profile.Phone,
		&profile.CurrentAddress, &profile.MotherMaidenName, &profile.Occupation,
		&profile.PlaceOfBirth, &profile.KTPImageID, &profile.KTPImageURL,
		&profile.SelfieImageID, &profile.SelfieImageURL, &profile.PayslipImageID,
		&profile.PayslipImageURL, &profile.AccountNumber, &profile.AccountHolderName,
		&profile.CurrentTotalDebt, &profile.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return profile, err
}
