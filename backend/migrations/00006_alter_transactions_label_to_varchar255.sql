-- Widens transactions.label from VARCHAR(50) to VARCHAR(255) to match
-- transaction_templates.label and prevent SQLSTATE 22001 ("value too long for
-- type character varying(50)") errors when users save labels longer than 50
-- characters from the new-transaction form.
--
-- The Down migration is DESTRUCTIVE: it truncates existing label values longer
-- than 50 characters before re-narrowing the column back to VARCHAR(50).
-- Without truncation, the ALTER TYPE would fail on any row whose label is
-- longer than 50 characters, leaving the migration unrevertable. The data loss
-- is intentional and accepted; this Down step exists for rollback completeness
-- only.

-- +goose Up
ALTER TABLE transactions ALTER COLUMN label TYPE VARCHAR(255);

-- +goose Down
UPDATE transactions
   SET label = LEFT(label, 50)
 WHERE label IS NOT NULL
   AND length(label) > 50;
ALTER TABLE transactions ALTER COLUMN label TYPE VARCHAR(50);
