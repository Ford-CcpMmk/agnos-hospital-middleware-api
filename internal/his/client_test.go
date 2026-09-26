package his

import (
	"context"
	"errors"
	"testing"

	"agnos-assignment/internal/model"
)

type clientStub func(context.Context, model.Hospital, string) (model.Patient, error)

func (fn clientStub) SearchByID(ctx context.Context, hospital model.Hospital, id string) (model.Patient, error) {
	return fn(ctx, hospital, id)
}

func TestRegistryRoutesToHospitalClient(t *testing.T) {
	registry := NewRegistry(map[string]Client{
		"hospital-a": clientStub(func(_ context.Context, hospital model.Hospital, id string) (model.Patient, error) {
			if hospital.Code != "hospital-a" || id != "1103700123456" {
				t.Fatalf("unexpected request: %+v id=%q", hospital, id)
			}
			return model.Patient{PatientHN: "HN001"}, nil
		}),
	})

	patient, err := registry.SearchByID(context.Background(), model.Hospital{Code: "hospital-a"}, "1103700123456")
	if err != nil || patient.PatientHN != "HN001" {
		t.Fatalf("unexpected result: patient=%+v err=%v", patient, err)
	}
}

func TestRegistryReturnsExpectedErrors(t *testing.T) {
	registry := NewRegistry(map[string]Client{})
	_, err := registry.SearchByID(context.Background(), model.Hospital{Code: "hospital-b"}, "id")
	if !errors.Is(err, ErrUnsupportedHospital) {
		t.Fatalf("expected unsupported hospital, got %v", err)
	}

	registry = NewRegistry(map[string]Client{
		"hospital-a": clientStub(func(context.Context, model.Hospital, string) (model.Patient, error) {
			return model.Patient{}, ErrPatientNotFound
		}),
	})
	_, err = registry.SearchByID(context.Background(), model.Hospital{Code: "hospital-a"}, "id")
	if !errors.Is(err, ErrPatientNotFound) {
		t.Fatalf("expected wrapped HIS error, got %v", err)
	}
}
