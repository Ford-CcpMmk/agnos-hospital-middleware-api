package repository

import (
	"context"
	"errors"
	"fmt"

	"agnos-assignment/internal/model"

	"github.com/jackc/pgx/v5"
)

type HospitalRepository struct {
	db DBTX
}

func NewHospitalRepository(db DBTX) *HospitalRepository {
	return &HospitalRepository{db: db}
}

func (r *HospitalRepository) FindByID(ctx context.Context, id int64) (model.Hospital, error) {
	const query = `
		SELECT id, code, name, api_base_url, created_at, updated_at
		FROM hospitals
		WHERE id = $1
	`

	return r.scanHospital(r.db.QueryRow(ctx, query, id))
}

func (r *HospitalRepository) FindByCode(ctx context.Context, code string) (model.Hospital, error) {
	const query = `
		SELECT id, code, name, api_base_url, created_at, updated_at
		FROM hospitals
		WHERE code = $1
	`

	return r.scanHospital(r.db.QueryRow(ctx, query, code))
}

func (r *HospitalRepository) scanHospital(row pgx.Row) (model.Hospital, error) {
	var hospital model.Hospital
	err := row.Scan(
		&hospital.ID,
		&hospital.Code,
		&hospital.Name,
		&hospital.APIBaseURL,
		&hospital.CreatedAt,
		&hospital.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Hospital{}, ErrHospitalNotFound
	}
	if err != nil {
		return model.Hospital{}, fmt.Errorf("find hospital: %w", err)
	}

	return hospital, nil
}
