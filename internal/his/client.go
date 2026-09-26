package his

import (
	"context"
	"errors"
	"fmt"

	"agnos-assignment/internal/model"
)

var (
	ErrPatientNotFound     = errors.New("patient not found in HIS")
	ErrUnsupportedHospital = errors.New("unsupported hospital integration")
)

type Client interface {
	SearchByID(context.Context, model.Hospital, string) (model.Patient, error)
}

type Registry struct {
	clients map[string]Client
}

func NewRegistry(clients map[string]Client) *Registry {
	return &Registry{clients: clients}
}

func (r *Registry) SearchByID(
	ctx context.Context,
	hospital model.Hospital,
	id string,
) (model.Patient, error) {
	client, ok := r.clients[hospital.Code]
	if !ok {
		return model.Patient{}, ErrUnsupportedHospital
	}

	patient, err := client.SearchByID(ctx, hospital, id)
	if err != nil {
		return model.Patient{}, fmt.Errorf("search %s HIS: %w", hospital.Code, err)
	}

	return patient, nil
}
