-- +goose Up

-- The assignment does not provide a live Hospital A endpoint. In the local
-- Compose stack, this address reaches the Django mock over the Docker network.
UPDATE hospitals
SET api_base_url = 'http://hospital-a-mock:8000'
WHERE code = 'hospital-a';

-- +goose Down

UPDATE hospitals
SET api_base_url = 'https://hospital-a.api.co.th'
WHERE code = 'hospital-a';
