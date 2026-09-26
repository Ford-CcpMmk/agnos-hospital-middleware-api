package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"agnos-assignment/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type dbStub struct {
	row      pgx.Row
	rows     pgx.Rows
	queryErr error
	query    string
	args     []any
}

func (db *dbStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.query, db.args = query, args
	return db.row
}

func (db *dbStub) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	db.query, db.args = query, args
	return db.rows, db.queryErr
}

type rowStub struct {
	values []any
	err    error
}

func (row rowStub) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	return scanValues(dest, row.values)
}

type rowsStub struct {
	rows   [][]any
	index  int
	err    error
	closed bool
}

func (rows *rowsStub) Close()                                       { rows.closed = true }
func (rows *rowsStub) Err() error                                   { return rows.err }
func (rows *rowsStub) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (rows *rowsStub) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (rows *rowsStub) Values() ([]any, error)                       { return rows.rows[rows.index], nil }
func (rows *rowsStub) RawValues() [][]byte                          { return nil }
func (rows *rowsStub) Conn() *pgx.Conn                              { return nil }
func (rows *rowsStub) TypeMap() *pgtype.Map                         { return nil }

func (rows *rowsStub) Next() bool {
	rows.index++
	return rows.index < len(rows.rows)
}

func (rows *rowsStub) Scan(dest ...any) error {
	if rows.index < 0 || rows.index >= len(rows.rows) {
		return errors.New("scan called without a current row")
	}
	return scanValues(dest, rows.rows[rows.index])
}

func scanValues(dest, values []any) error {
	if len(dest) != len(values) {
		return errors.New("unexpected scan destination count")
	}

	for index, value := range values {
		target := reflect.ValueOf(dest[index])
		if target.Kind() != reflect.Ptr || target.IsNil() {
			return errors.New("scan destination must be a non-nil pointer")
		}
		if value == nil {
			target.Elem().SetZero()
			continue
		}

		source := reflect.ValueOf(value)
		if !source.Type().AssignableTo(target.Elem().Type()) {
			return errors.New("unexpected scan value type")
		}
		target.Elem().Set(source)
	}

	return nil
}

func TestHospitalRepositoryFindByID(t *testing.T) {
	createdAt := time.Date(2026, time.September, 26, 0, 0, 0, 0, time.UTC)
	apiURL := "https://hospital-a.example"
	db := &dbStub{row: rowStub{values: []any{int64(7), "hospital-a", "Hospital A", &apiURL, createdAt, createdAt}}}

	hospital, err := NewHospitalRepository(db).FindByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("find hospital: %v", err)
	}
	if hospital.ID != 7 || hospital.Code != "hospital-a" || hospital.APIBaseURL == nil || *hospital.APIBaseURL != apiURL {
		t.Fatalf("unexpected hospital: %+v", hospital)
	}
	if !strings.Contains(db.query, "WHERE id = $1") || !reflect.DeepEqual(db.args, []any{int64(7)}) {
		t.Fatalf("unexpected query: %q args=%v", db.query, db.args)
	}
}

func TestHospitalRepositoryMapsMissingHospital(t *testing.T) {
	db := &dbStub{row: rowStub{err: pgx.ErrNoRows}}
	_, err := NewHospitalRepository(db).FindByCode(context.Background(), "unknown")
	if !errors.Is(err, ErrHospitalNotFound) {
		t.Fatalf("expected hospital not found, got %v", err)
	}
}

func TestStaffRepositoryCreateAndFind(t *testing.T) {
	now := time.Date(2026, time.September, 26, 0, 0, 0, 0, time.UTC)
	db := &dbStub{row: rowStub{values: []any{int64(11), int64(7), "staff01", now, now}}}
	repo := NewStaffRepository(db)

	created, err := repo.Create(context.Background(), model.Staff{HospitalID: 7, Username: "staff01", PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}
	if created.ID != 11 || created.Username != "staff01" || created.PasswordHash != "" {
		t.Fatalf("unexpected created staff: %+v", created)
	}
	if !strings.Contains(db.query, "INSERT INTO staff") || !reflect.DeepEqual(db.args, []any{int64(7), "staff01", "hash"}) {
		t.Fatalf("unexpected create query: %q args=%v", db.query, db.args)
	}

	db.row = rowStub{values: []any{int64(11), int64(7), "staff01", "hash", now, now}}
	staff, err := repo.FindByUsername(context.Background(), 7, "staff01")
	if err != nil {
		t.Fatalf("find staff: %v", err)
	}
	if staff.PasswordHash != "hash" || !strings.Contains(db.query, "WHERE hospital_id = $1 AND username = $2") {
		t.Fatalf("unexpected staff or lookup query: %+v; %q", staff, db.query)
	}
}

func TestStaffRepositoryMapsExpectedErrors(t *testing.T) {
	missingDB := &dbStub{row: rowStub{err: pgx.ErrNoRows}}
	_, err := NewStaffRepository(missingDB).FindByUsername(context.Background(), 7, "unknown")
	if !errors.Is(err, ErrStaffNotFound) {
		t.Fatalf("expected staff not found, got %v", err)
	}

	duplicateDB := &dbStub{row: rowStub{err: &pgconn.PgError{Code: uniqueViolationCode}}}
	_, err = NewStaffRepository(duplicateDB).Create(context.Background(), model.Staff{})
	if !errors.Is(err, ErrStaffAlreadyExists) {
		t.Fatalf("expected staff already exists, got %v", err)
	}
}

func TestPatientRepositorySearchAndUpsert(t *testing.T) {
	now := time.Date(2026, time.September, 26, 0, 0, 0, 0, time.UTC)
	firstName := "Som%chai"
	nationalID := "1103700123456"
	gender := model.GenderMale
	patientValues := []any{
		int64(21), int64(7), "HN001", &nationalID, nil,
		(*string)(nil), (*string)(nil), (*string)(nil), &firstName, (*string)(nil), (*string)(nil),
		(*time.Time)(nil), (*string)(nil), (*string)(nil), &gender, now, now,
	}
	db := &dbStub{rows: &rowsStub{index: -1, rows: [][]any{append(patientValues, int64(1))}}}
	repo := NewPatientRepository(db)

	patients, total, err := repo.Search(context.Background(), PatientSearchFilter{
		HospitalID: 7,
		FirstName:  &firstName,
		Limit:      20,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("search patients: %v", err)
	}
	if total != 1 || len(patients) != 1 || patients[0].PatientHN != "HN001" {
		t.Fatalf("unexpected patients: total=%d patients=%+v", total, patients)
	}
	if !strings.Contains(db.query, "ILIKE $2 ESCAPE '\\'") || !reflect.DeepEqual(db.args, []any{int64(7), "%Som\\%chai%", 20, 0}) {
		t.Fatalf("unexpected search query: %q args=%v", db.query, db.args)
	}

	db.row = rowStub{values: patientValues}
	saved, err := repo.Upsert(context.Background(), model.Patient{HospitalID: 7, PatientHN: "HN001", NationalID: &nationalID})
	if err != nil {
		t.Fatalf("upsert patient: %v", err)
	}
	if saved.ID != 21 || saved.HospitalID != 7 || !strings.Contains(db.query, "ON CONFLICT (hospital_id, patient_hn)") {
		t.Fatalf("unexpected saved patient or query: %+v; %q", saved, db.query)
	}
}

func TestPatientRepositoryReturnsQueryError(t *testing.T) {
	db := &dbStub{queryErr: errors.New("database unavailable")}
	_, _, err := NewPatientRepository(db).Search(context.Background(), PatientSearchFilter{HospitalID: 7, Limit: 20})
	if err == nil || !strings.Contains(err.Error(), "search patients") {
		t.Fatalf("expected wrapped search error, got %v", err)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`a%_b\c`); got != `a\%\_b\\c` {
		t.Fatalf("unexpected escaped LIKE value: %q", got)
	}
}
