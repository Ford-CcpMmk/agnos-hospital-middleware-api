package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agnos-assignment/internal/model"
)

type PatientSearchFilter struct {
	HospitalID  int64
	NationalID  *string
	PassportID  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	DateOfBirth *time.Time
	PhoneNumber *string
	Email       *string
	Limit       int
	Offset      int
}

type PatientRepository struct {
	db DBTX
}

func NewPatientRepository(db DBTX) *PatientRepository {
	return &PatientRepository{db: db}
}

func (r *PatientRepository) Search(
	ctx context.Context,
	filter PatientSearchFilter,
) ([]model.Patient, int64, error) {
	conditions := []string{"hospital_id = $1"}
	args := []any{filter.HospitalID}

	addExactCondition := func(column string, value *string) {
		if value == nil {
			return
		}
		args = append(args, *value)
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	addNameCondition := func(thaiColumn, englishColumn string, value *string) {
		if value == nil {
			return
		}
		args = append(args, "%"+escapeLike(*value)+"%")
		position := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"(%s ILIKE $%d ESCAPE '\\' OR %s ILIKE $%d ESCAPE '\\')",
			thaiColumn,
			position,
			englishColumn,
			position,
		))
	}

	addExactCondition("national_id", filter.NationalID)
	addExactCondition("passport_id", filter.PassportID)
	addNameCondition("first_name_th", "first_name_en", filter.FirstName)
	addNameCondition("middle_name_th", "middle_name_en", filter.MiddleName)
	addNameCondition("last_name_th", "last_name_en", filter.LastName)
	if filter.DateOfBirth != nil {
		args = append(args, *filter.DateOfBirth)
		conditions = append(conditions, fmt.Sprintf("date_of_birth = $%d", len(args)))
	}
	addExactCondition("phone_number", filter.PhoneNumber)
	addExactCondition("email", filter.Email)

	args = append(args, filter.Limit)
	limitPosition := len(args)
	args = append(args, filter.Offset)
	offsetPosition := len(args)

	query := fmt.Sprintf(`
		SELECT
			id, hospital_id, patient_hn, national_id, passport_id,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, phone_number, email, gender,
			created_at, updated_at, COUNT(*) OVER()
		FROM patients
		WHERE %s
		ORDER BY id
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), limitPosition, offsetPosition)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("search patients: %w", err)
	}
	defer rows.Close()

	patients := make([]model.Patient, 0)
	var total int64
	for rows.Next() {
		var patient model.Patient
		if err := rows.Scan(
			&patient.ID,
			&patient.HospitalID,
			&patient.PatientHN,
			&patient.NationalID,
			&patient.PassportID,
			&patient.FirstNameTH,
			&patient.MiddleNameTH,
			&patient.LastNameTH,
			&patient.FirstNameEN,
			&patient.MiddleNameEN,
			&patient.LastNameEN,
			&patient.DateOfBirth,
			&patient.PhoneNumber,
			&patient.Email,
			&patient.Gender,
			&patient.CreatedAt,
			&patient.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan patient search result: %w", err)
		}
		patients = append(patients, patient)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate patient search results: %w", err)
	}

	return patients, total, nil
}

func (r *PatientRepository) Upsert(ctx context.Context, patient model.Patient) (model.Patient, error) {
	const query = `
		INSERT INTO patients (
			hospital_id, patient_hn, national_id, passport_id,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, phone_number, email, gender
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (hospital_id, patient_hn) DO UPDATE SET
			national_id = EXCLUDED.national_id,
			passport_id = EXCLUDED.passport_id,
			first_name_th = EXCLUDED.first_name_th,
			middle_name_th = EXCLUDED.middle_name_th,
			last_name_th = EXCLUDED.last_name_th,
			first_name_en = EXCLUDED.first_name_en,
			middle_name_en = EXCLUDED.middle_name_en,
			last_name_en = EXCLUDED.last_name_en,
			date_of_birth = EXCLUDED.date_of_birth,
			phone_number = EXCLUDED.phone_number,
			email = EXCLUDED.email,
			gender = EXCLUDED.gender,
			updated_at = NOW()
		RETURNING
			id, hospital_id, patient_hn, national_id, passport_id,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, phone_number, email, gender, created_at, updated_at
	`

	var saved model.Patient
	err := r.db.QueryRow(
		ctx,
		query,
		patient.HospitalID,
		patient.PatientHN,
		patient.NationalID,
		patient.PassportID,
		patient.FirstNameTH,
		patient.MiddleNameTH,
		patient.LastNameTH,
		patient.FirstNameEN,
		patient.MiddleNameEN,
		patient.LastNameEN,
		patient.DateOfBirth,
		patient.PhoneNumber,
		patient.Email,
		patient.Gender,
	).Scan(
		&saved.ID,
		&saved.HospitalID,
		&saved.PatientHN,
		&saved.NationalID,
		&saved.PassportID,
		&saved.FirstNameTH,
		&saved.MiddleNameTH,
		&saved.LastNameTH,
		&saved.FirstNameEN,
		&saved.MiddleNameEN,
		&saved.LastNameEN,
		&saved.DateOfBirth,
		&saved.PhoneNumber,
		&saved.Email,
		&saved.Gender,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if err != nil {
		return model.Patient{}, fmt.Errorf("upsert patient: %w", err)
	}

	return saved, nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return replacer.Replace(value)
}
