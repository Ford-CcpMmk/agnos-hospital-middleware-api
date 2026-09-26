package repository

import (
	"context"
	"errors"
	"fmt"

	"agnos-assignment/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

type StaffRepository struct {
	db DBTX
}

func NewStaffRepository(db DBTX) *StaffRepository {
	return &StaffRepository{db: db}
}

func (r *StaffRepository) FindByUsername(
	ctx context.Context,
	hospitalID int64,
	username string,
) (model.Staff, error) {
	const query = `
		SELECT id, hospital_id, username, password_hash, created_at, updated_at
		FROM staff
		WHERE hospital_id = $1 AND username = $2
	`

	var staff model.Staff
	err := r.db.QueryRow(ctx, query, hospitalID, username).Scan(
		&staff.ID,
		&staff.HospitalID,
		&staff.Username,
		&staff.PasswordHash,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Staff{}, ErrStaffNotFound
	}
	if err != nil {
		return model.Staff{}, fmt.Errorf("find staff by username: %w", err)
	}

	return staff, nil
}

func (r *StaffRepository) Create(ctx context.Context, staff model.Staff) (model.Staff, error) {
	const query = `
		INSERT INTO staff (hospital_id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, hospital_id, username, created_at, updated_at
	`

	var created model.Staff
	err := r.db.QueryRow(
		ctx,
		query,
		staff.HospitalID,
		staff.Username,
		staff.PasswordHash,
	).Scan(
		&created.ID,
		&created.HospitalID,
		&created.Username,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err == nil {
		return created, nil
	}

	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == uniqueViolationCode {
		return model.Staff{}, ErrStaffAlreadyExists
	}

	return model.Staff{}, fmt.Errorf("create staff: %w", err)
}
