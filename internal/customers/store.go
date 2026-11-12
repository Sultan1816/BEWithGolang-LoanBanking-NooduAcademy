package customers

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("customer tidak ditemukan")

type Customer struct {
	ID                  int64
	User_ID             int64
	NIK                 string
	Address             string
	Date_0f_Birth       time.Time
	Monthly_Income      float64
	Phone               string
	Current_Address     string
	Mother_Maiden_Name  string
	Occupation          string
	Place_of_Birth      string
	KTP_Image_Path      string
	Selfie_Image_Path   string
	Payslip_Image_Path  string
	Account_Number      string
	Account_Holder_Name string
	Current_Total_Debt  float64
	Created_At          time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (store *Store) ByUserID(ctx context.Context, user_id int64) (Customer, error) {
	var customer Customer
	err := store.db.QueryRowContext(ctx,
		`SELECT id, user_id, nik, COALESCE(address, ''), COALESCE(date_of_birth, DATE '0001-01-01'),
		        COALESCE(monthly_income, 0), COALESCE(phone, ''), COALESCE(current_address, ''),
		        COALESCE(mother_maiden_name, ''), COALESCE(occupation, ''), COALESCE(place_of_birth, ''),
		        COALESCE(ktp_image_path, ''), COALESCE(selfie_image_path, ''), COALESCE(payslip_image_path, ''),
		        COALESCE(account_number, ''), COALESCE(account_holder_name, ''),
		        COALESCE(current_total_debt, 0), COALESCE(created_at, TIMESTAMP '0001-01-01')
		 FROM customers WHERE user_id = $1`,
		user_id).Scan(&customer.ID, &customer.User_ID, &customer.NIK, &customer.Address, &customer.Date_0f_Birth, &customer.Monthly_Income, &customer.Phone, &customer.Current_Address, &customer.Mother_Maiden_Name, &customer.Occupation, &customer.Place_of_Birth, &customer.KTP_Image_Path, &customer.Selfie_Image_Path, &customer.Payslip_Image_Path, &customer.Account_Number, &customer.Account_Holder_Name, &customer.Current_Total_Debt, &customer.Created_At)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Customer{}, ErrNotFound
		}
		return customer, err
	}
	return customer, nil
}

func (store *Store) ByID(ctx context.Context, id int64) (Customer, error) {
	var customer Customer
	err := store.db.QueryRowContext(ctx,
		`SELECT id, user_id, nik, COALESCE(address, ''), COALESCE(date_of_birth, DATE '0001-01-01'),
		        COALESCE(monthly_income, 0), COALESCE(phone, ''), COALESCE(current_address, ''),
		        COALESCE(mother_maiden_name, ''), COALESCE(occupation, ''), COALESCE(place_of_birth, ''),
		        COALESCE(ktp_image_path, ''), COALESCE(selfie_image_path, ''), COALESCE(payslip_image_path, ''),
		        COALESCE(account_number, ''), COALESCE(account_holder_name, ''),
		        COALESCE(current_total_debt, 0), COALESCE(created_at, TIMESTAMP '0001-01-01')
		 FROM customers WHERE id = $1`,
		id).Scan(&customer.ID, &customer.User_ID, &customer.NIK, &customer.Address, &customer.Date_0f_Birth, &customer.Monthly_Income, &customer.Phone, &customer.Current_Address, &customer.Mother_Maiden_Name, &customer.Occupation, &customer.Place_of_Birth, &customer.KTP_Image_Path, &customer.Selfie_Image_Path, &customer.Payslip_Image_Path, &customer.Account_Number, &customer.Account_Holder_Name, &customer.Current_Total_Debt, &customer.Created_At)
	if errors.Is(err, sql.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	return customer, err
}

func (store *Store) Create(ctx context.Context, user_id int64, nik, address string, date_of_birth time.Time, monthly_income float64, phone, current_address, mother_maiden_name, occupation, place_of_birth, ktp_image_path, selfie_image_path, payslip_image_path, account_number, account_holder_name string) (Customer, error) {
	var customer Customer
	err := store.db.QueryRowContext(ctx,
		`INSERT INTO customers (
		user_id, nik, address, date_of_birth, monthly_income, phone, current_address, mother_maiden_name, occupation, place_of_birth, ktp_image_path, selfie_image_path, payslip_image_path, account_number, account_holder_name) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id, user_id, nik, address, date_of_birth, monthly_income, phone, current_address, mother_maiden_name, occupation, place_of_birth, ktp_image_path, selfie_image_path, payslip_image_path, account_number, account_holder_name, current_total_debt, created_at`,
		user_id, strings.TrimSpace(nik), strings.TrimSpace(address), date_of_birth, monthly_income, strings.TrimSpace(phone), strings.TrimSpace(current_address), strings.TrimSpace(mother_maiden_name), strings.TrimSpace(occupation), strings.TrimSpace(place_of_birth), strings.TrimSpace(ktp_image_path), strings.TrimSpace(selfie_image_path), strings.TrimSpace(payslip_image_path), strings.TrimSpace(account_number), strings.TrimSpace(account_holder_name),
	).Scan(&customer.ID, &customer.User_ID, &customer.NIK, &customer.Address, &customer.Date_0f_Birth, &customer.Monthly_Income, &customer.Phone, &customer.Current_Address, &customer.Mother_Maiden_Name, &customer.Occupation, &customer.Place_of_Birth, &customer.KTP_Image_Path, &customer.Selfie_Image_Path, &customer.Payslip_Image_Path, &customer.Account_Number, &customer.Account_Holder_Name, &customer.Current_Total_Debt, &customer.Created_At)
	return customer, err
}

func (store *Store) UpdateProfile(ctx context.Context, id int64, nik, address *string, date_of_birth *time.Time, monthly_income *float64, phone, current_address, mother_maiden_name, occupation, place_of_birth, ktp_image_path, selfie_image_path, payslip_image_path, account_number, account_holder_name *string) (Customer, error) {
	var (
		nikValue, addressValue, phoneValue, currentAddressValue        string
		motherMaidenNameValue, occupationValue, placeOfBirthValue      string
		ktpImagePathValue, selfieImagePathValue, payslipImagePathValue string
		accountNumberValue, accountHolderNameValue                     string
		dateOfBirthValue                                               time.Time
		monthlyIncomeValue                                             float64
	)
	if nik != nil {
		nikValue = strings.TrimSpace(*nik)
	}
	if address != nil {
		addressValue = strings.TrimSpace(*address)
	}
	if date_of_birth != nil {
		dateOfBirthValue = *date_of_birth
	}
	if monthly_income != nil {
		monthlyIncomeValue = *monthly_income
	}
	if phone != nil {
		phoneValue = strings.TrimSpace(*phone)
	}
	if current_address != nil {
		currentAddressValue = strings.TrimSpace(*current_address)
	}
	if mother_maiden_name != nil {
		motherMaidenNameValue = strings.TrimSpace(*mother_maiden_name)
	}
	if occupation != nil {
		occupationValue = strings.TrimSpace(*occupation)
	}
	if place_of_birth != nil {
		placeOfBirthValue = strings.TrimSpace(*place_of_birth)
	}
	if ktp_image_path != nil {
		ktpImagePathValue = strings.TrimSpace(*ktp_image_path)
	}
	if selfie_image_path != nil {
		selfieImagePathValue = strings.TrimSpace(*selfie_image_path)
	}
	if payslip_image_path != nil {
		payslipImagePathValue = strings.TrimSpace(*payslip_image_path)
	}
	if account_number != nil {
		accountNumberValue = strings.TrimSpace(*account_number)
	}
	if account_holder_name != nil {
		accountHolderNameValue = strings.TrimSpace(*account_holder_name)
	}

	var customer Customer
	err := store.db.QueryRowContext(ctx,
		`UPDATE customers
		 SET nik = CASE WHEN $2 THEN $3 ELSE nik END,
		     address = CASE WHEN $4 THEN $5 ELSE address END,
		     date_of_birth = CASE WHEN $6 THEN $7 ELSE date_of_birth END,
		     monthly_income = CASE WHEN $8 THEN $9 ELSE monthly_income END,
		     phone = CASE WHEN $10 THEN $11 ELSE phone END,
		     current_address = CASE WHEN $12 THEN $13 ELSE current_address END,
		     mother_maiden_name = CASE WHEN $14 THEN $15 ELSE mother_maiden_name END,
		     occupation = CASE WHEN $16 THEN $17 ELSE occupation END,
		     place_of_birth = CASE WHEN $18 THEN $19 ELSE place_of_birth END,
		     ktp_image_path = CASE WHEN $20 THEN $21 ELSE ktp_image_path END,
		     selfie_image_path = CASE WHEN $22 THEN $23 ELSE selfie_image_path END,
		     payslip_image_path = CASE WHEN $24 THEN $25 ELSE payslip_image_path END,
		     account_number = CASE WHEN $26 THEN $27 ELSE account_number END,
		     account_holder_name = CASE WHEN $28 THEN $29 ELSE account_holder_name END
		 WHERE id = $1
		 RETURNING id, user_id, nik, COALESCE(address, ''), COALESCE(date_of_birth, DATE '0001-01-01'),
		           COALESCE(monthly_income, 0), COALESCE(phone, ''), COALESCE(current_address, ''),
		           COALESCE(mother_maiden_name, ''), COALESCE(occupation, ''), COALESCE(place_of_birth, ''),
		           COALESCE(ktp_image_path, ''), COALESCE(selfie_image_path, ''), COALESCE(payslip_image_path, ''),
		           COALESCE(account_number, ''), COALESCE(account_holder_name, ''),
		           COALESCE(current_total_debt, 0), COALESCE(created_at, TIMESTAMP '0001-01-01')`,
		id, nik != nil, nikValue, address != nil, addressValue, date_of_birth != nil, dateOfBirthValue, monthly_income != nil, monthlyIncomeValue, phone != nil, phoneValue, current_address != nil, currentAddressValue, mother_maiden_name != nil, motherMaidenNameValue, occupation != nil, occupationValue, place_of_birth != nil, placeOfBirthValue, ktp_image_path != nil, ktpImagePathValue, selfie_image_path != nil, selfieImagePathValue, payslip_image_path != nil, payslipImagePathValue, account_number != nil, accountNumberValue, account_holder_name != nil, accountHolderNameValue,
	).Scan(&customer.ID, &customer.User_ID, &customer.NIK, &customer.Address, &customer.Date_0f_Birth, &customer.Monthly_Income, &customer.Phone, &customer.Current_Address, &customer.Mother_Maiden_Name, &customer.Occupation, &customer.Place_of_Birth, &customer.KTP_Image_Path, &customer.Selfie_Image_Path, &customer.Payslip_Image_Path, &customer.Account_Number, &customer.Account_Holder_Name, &customer.Current_Total_Debt, &customer.Created_At)
	if errors.Is(err, sql.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	return customer, err
}

func (store *Store) Delete(ctx context.Context, id int64) error {
	result, err := store.db.ExecContext(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
