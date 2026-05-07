-- +goose Up
CREATE TABLE plan_prices (
    id              SERIAL PRIMARY KEY,
    plan_id         INT NOT NULL REFERENCES subscription_plans(id),
    provider_type   TEXT NOT NULL,
    external_price_id TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE plan_prices
    ADD CONSTRAINT uq_plan_prices_plan_id_provider_type UNIQUE (plan_id, provider_type);

CREATE INDEX idx_plan_prices_provider_type ON plan_prices (provider_type);

-- +goose Down
DROP TABLE IF EXISTS plan_prices;
