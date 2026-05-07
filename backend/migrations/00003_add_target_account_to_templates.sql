-- +goose Up
ALTER TABLE transaction_templates
    ADD COLUMN target_account_id INT NULL REFERENCES accounts(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE transaction_templates
    DROP COLUMN IF EXISTS target_account_id;
