package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"noodu-login-service/internal/customerprofile"
	"noodu-login-service/internal/customers"
	"noodu-login-service/internal/users"
)

const (
	maxProfileRequestSize = 26 << 20
	maxProfileImageSize   = 8 << 20
)

func (server *Server) createCustomerProfile(w http.ResponseWriter, request *http.Request) {
	input, documents, err := parseCustomerProfileForm(w, request)
	if err != nil {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	defer request.MultipartForm.RemoveAll()

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

	profile, err := server.customerProfiles.Create(request.Context(), userID, input, documents)
	if errors.Is(err, customerprofile.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid_profile", "NIK harus 16 digit, tanggal lahir harus YYYY-MM-DD, dan penghasilan tidak boleh negatif")
		return
	}
	if errors.Is(err, customerprofile.ErrInvalidImage) {
		writeError(w, http.StatusBadRequest, "invalid_image", "KTP, selfie, dan slip gaji wajib berupa gambar JPEG atau PNG dengan ukuran maksimal 8 MiB per gambar")
		return
	}
	if errors.Is(err, customerprofile.ErrProfileExists) || isUniqueViolation(err) {
		writeError(w, http.StatusConflict, "profile_exists", "Profil customer atau NIK sudah terdaftar")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Profil customer tidak dapat disimpan")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": profile})
}

func (server *Server) getCustomerProfile(w http.ResponseWriter, request *http.Request) {
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

	profile, err := server.customerProfiles.ByUserID(request.Context(), userID)
	if errors.Is(err, customers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile_not_found", "Profil customer belum diisi")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Profil customer tidak dapat dibaca")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": profile})
}

func parseCustomerProfileForm(w http.ResponseWriter, request *http.Request) (customerprofile.CreateInput, customerprofile.Documents, error) {
	if !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data") {
		return customerprofile.CreateInput{}, customerprofile.Documents{}, fmt.Errorf("Content-Type harus multipart/form-data")
	}
	request.Body = http.MaxBytesReader(w, request.Body, maxProfileRequestSize)
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		return customerprofile.CreateInput{}, customerprofile.Documents{}, fmt.Errorf("form multipart tidak valid atau ukuran request terlalu besar")
	}

	input := customerprofile.CreateInput{
		NIK:               request.FormValue("nik"),
		Address:           optionalFormValue(request, "address"),
		DateOfBirth:       optionalFormValue(request, "date_of_birth"),
		Phone:             optionalFormValue(request, "phone"),
		CurrentAddress:    optionalFormValue(request, "current_address"),
		MotherMaidenName:  optionalFormValue(request, "mother_maiden_name"),
		Occupation:        optionalFormValue(request, "occupation"),
		PlaceOfBirth:      optionalFormValue(request, "place_of_birth"),
		AccountNumber:     optionalFormValue(request, "account_number"),
		AccountHolderName: optionalFormValue(request, "account_holder_name"),
	}
	if value := request.FormValue("monthly_income"); value != "" {
		income, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return customerprofile.CreateInput{}, customerprofile.Documents{}, fmt.Errorf("monthly_income harus berupa angka")
		}
		input.MonthlyIncome = &income
	}

	ktp, err := readProfileImage(request.MultipartForm, "ktp_image")
	if err != nil {
		return customerprofile.CreateInput{}, customerprofile.Documents{}, err
	}
	selfie, err := readProfileImage(request.MultipartForm, "selfie_image")
	if err != nil {
		return customerprofile.CreateInput{}, customerprofile.Documents{}, err
	}
	payslip, err := readProfileImage(request.MultipartForm, "payslip_image")
	if err != nil {
		return customerprofile.CreateInput{}, customerprofile.Documents{}, err
	}
	return input, customerprofile.Documents{KTP: ktp, Selfie: selfie, Payslip: payslip}, nil
}

func optionalFormValue(request *http.Request, key string) *string {
	_, exists := request.Form[key]
	if !exists {
		return nil
	}
	trimmed := strings.TrimSpace(request.FormValue(key))
	return &trimmed
}

func readProfileImage(form *multipart.Form, field string) (customerprofile.ImageUpload, error) {
	if form == nil || len(form.File[field]) != 1 {
		return customerprofile.ImageUpload{}, fmt.Errorf("%s wajib diisi dengan satu file gambar", field)
	}
	header := form.File[field][0]
	if header.Size <= 0 || header.Size > maxProfileImageSize {
		return customerprofile.ImageUpload{}, fmt.Errorf("%s harus berukuran maksimal 8 MiB", field)
	}
	file, err := header.Open()
	if err != nil {
		return customerprofile.ImageUpload{}, fmt.Errorf("file %s tidak dapat dibaca", field)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxProfileImageSize+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(data) > maxProfileImageSize {
		return customerprofile.ImageUpload{}, fmt.Errorf("file %s tidak dapat dibaca atau terlalu besar", field)
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return customerprofile.ImageUpload{}, fmt.Errorf("%s harus berupa gambar JPEG atau PNG", field)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") ||
		config.Width <= 0 || config.Height <= 0 ||
		config.Width > 10000 || config.Height > 10000 ||
		int64(config.Width)*int64(config.Height) > 40000000 {
		return customerprofile.ImageUpload{}, fmt.Errorf("%s bukan gambar JPEG/PNG yang valid atau dimensinya terlalu besar", field)
	}
	return customerprofile.ImageUpload{
		Filename:    header.Filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}
