-- +goose Up
-- +goose StatementBegin

CREATE TABLE hospitals (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    api_base_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE staff (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hospital_id BIGINT NOT NULL REFERENCES hospitals(id) ON DELETE RESTRICT,
    username VARCHAR(100) NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT staff_hospital_username_key UNIQUE (hospital_id, username)
);

CREATE INDEX staff_hospital_id_idx ON staff (hospital_id);

CREATE TABLE patients (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hospital_id BIGINT NOT NULL REFERENCES hospitals(id) ON DELETE RESTRICT,
    patient_hn VARCHAR(100) NOT NULL,
    national_id VARCHAR(20),
    passport_id VARCHAR(32),
    first_name_th VARCHAR(255),
    middle_name_th VARCHAR(255),
    last_name_th VARCHAR(255),
    first_name_en VARCHAR(255),
    middle_name_en VARCHAR(255),
    last_name_en VARCHAR(255),
    date_of_birth DATE,
    phone_number VARCHAR(32),
    email VARCHAR(320),
    gender CHAR(1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT patients_gender_check CHECK (gender IS NULL OR gender IN ('M', 'F')),
    CONSTRAINT patients_hospital_hn_key UNIQUE (hospital_id, patient_hn)
);

CREATE INDEX patients_hospital_id_idx ON patients (hospital_id);
CREATE INDEX patients_hospital_date_of_birth_idx
    ON patients (hospital_id, date_of_birth);
CREATE INDEX patients_hospital_email_idx
    ON patients (hospital_id, email);
CREATE INDEX patients_hospital_phone_number_idx
    ON patients (hospital_id, phone_number);

CREATE UNIQUE INDEX patients_hospital_national_id_key
    ON patients (hospital_id, national_id)
    WHERE national_id IS NOT NULL;

CREATE UNIQUE INDEX patients_hospital_passport_id_key
    ON patients (hospital_id, passport_id)
    WHERE passport_id IS NOT NULL;

INSERT INTO hospitals (code, name, api_base_url)
VALUES ('hospital-a', 'Hospital A', 'https://hospital-a.api.co.th');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS hospitals;

-- +goose StatementEnd
