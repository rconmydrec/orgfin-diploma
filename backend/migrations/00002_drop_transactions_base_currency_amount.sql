-- +goose Up
ALTER TABLE transactions DROP COLUMN IF EXISTS base_currency_amount;

-- +goose Down
ALTER TABLE transactions ADD COLUMN base_currency_amount NUMERIC;
