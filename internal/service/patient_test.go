package service_test

import (
	"context"
	"errors"
	"testing"

	"agnos-assignment/internal/his"
	"agnos-assignment/internal/model"
	"agnos-assignment/internal/repository"
	"agnos-assignment/internal/service"
)

type patientHospitalFinderFunc func(context.Context, int64) (model.Hospital, error)

func (fn patientHospitalFinderFunc) FindByID(ctx context.Context, id int64) (model.Hospital, error) {
	return fn(ctx, id)
}

type patientStoreStub struct {
	searchFn func(context.Context, repository.PatientSearchFilter) ([]model.Patient, int64, error)
	upsertFn func(context.Context, model.Patient) (model.Patient, error)
}

func (stub patientStoreStub) Search(
	ctx context.Context,
	filter repository.PatientSearchFilter,
) ([]model.Patient, int64, error) {
	return stub.searchFn(ctx, filter)
}

func (stub patientStoreStub) Upsert(ctx context.Context, patient model.Patient) (model.Patient, error) {
	return stub.upsertFn(ctx, patient)
}

type hisPatientFinderFunc func(context.Context, model.Hospital, string) (model.Patient, error)

func (fn hisPatientFinderFunc) SearchByID(
	ctx context.Context,
	hospital model.Hospital,
	id string,
) (model.Patient, error) {
	return fn(ctx, hospital, id)
}

func TestPatientServiceSearchUsesLocalHospitalScope(t *testing.T) {
	firstName := "Somchai"
	patientService := service.NewPatientService(
		nil,
		patientStoreStub{searchFn: func(_ context.Context, filter repository.PatientSearchFilter) ([]model.Patient, int64, error) {
			if filter.HospitalID != 7 || filter.FirstName == nil || *filter.FirstName != firstName {
				t.Fatalf("unexpected search filter: %+v", filter)
			}
			return []model.Patient{{ID: 1, HospitalID: 7, PatientHN: "HN001"}}, 1, nil
		}},
		nil,
	)

	result, err := patientService.Search(context.Background(), service.SearchPatientsInput{
		HospitalID: 7,
		FirstName:  &firstName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 1 || len(result.Patients) != 1 || result.Patients[0].HospitalID != 7 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPatientServiceSearchFallsBackToHISAndCachesPatient(t *testing.T) {
	nationalID := "1103700123456"
	patientFromHIS := model.Patient{PatientHN: "HN001", NationalID: &nationalID}

	patientService := service.NewPatientService(
		patientHospitalFinderFunc(func(_ context.Context, id int64) (model.Hospital, error) {
			return model.Hospital{ID: id, Code: "hospital-a"}, nil
		}),
		patientStoreStub{
			searchFn: func(context.Context, repository.PatientSearchFilter) ([]model.Patient, int64, error) {
				return []model.Patient{}, 0, nil
			},
			upsertFn: func(_ context.Context, patient model.Patient) (model.Patient, error) {
				if patient.HospitalID != 7 {
					t.Fatalf("expected hospital ID 7, got %d", patient.HospitalID)
				}
				patient.ID = 1
				return patient, nil
			},
		},
		hisPatientFinderFunc(func(_ context.Context, hospital model.Hospital, id string) (model.Patient, error) {
			if hospital.Code != "hospital-a" || id != nationalID {
				t.Fatalf("unexpected HIS request: hospital=%q id=%q", hospital.Code, id)
			}
			return patientFromHIS, nil
		}),
	)

	result, err := patientService.Search(context.Background(), service.SearchPatientsInput{
		HospitalID:   7,
		HospitalCode: "hospital-a",
		NationalID:   &nationalID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 1 || len(result.Patients) != 1 || result.Patients[0].ID != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPatientServiceSearchErrors(t *testing.T) {
	patientService := service.NewPatientService(
		nil,
		patientStoreStub{searchFn: func(context.Context, repository.PatientSearchFilter) ([]model.Patient, int64, error) {
			return nil, 0, errors.New("search should not be called")
		}},
		nil,
	)

	_, err := patientService.Search(context.Background(), service.SearchPatientsInput{HospitalID: 7})
	if !errors.Is(err, service.ErrNoSearchCriteria) {
		t.Fatalf("expected no-search-criteria error, got %v", err)
	}
}

func TestPatientServiceSearchReturnsEmptyWhenHISHasNoPatient(t *testing.T) {
	passportID := "AA123456"
	patientService := service.NewPatientService(
		patientHospitalFinderFunc(func(_ context.Context, id int64) (model.Hospital, error) {
			return model.Hospital{ID: id, Code: "hospital-a"}, nil
		}),
		patientStoreStub{searchFn: func(context.Context, repository.PatientSearchFilter) ([]model.Patient, int64, error) {
			return []model.Patient{}, 0, nil
		}},
		hisPatientFinderFunc(func(context.Context, model.Hospital, string) (model.Patient, error) {
			return model.Patient{}, his.ErrPatientNotFound
		}),
	)

	result, err := patientService.Search(context.Background(), service.SearchPatientsInput{
		HospitalID:   7,
		HospitalCode: "hospital-a",
		PassportID:   &passportID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 0 || len(result.Patients) != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
}
