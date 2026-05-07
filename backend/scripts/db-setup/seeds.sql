-- Seed data for budgeter test/dev database
-- Run after schema.sql has been applied
-- Uses ON CONFLICT DO NOTHING for idempotency

BEGIN;

-- ============================================================================
-- Currencies (34 rows)
-- ============================================================================
INSERT INTO currencies (id, code, name) VALUES
    (1, 'USD', 'United States Dollar'),
    (2, 'UAH', 'Ukrainian Hryvna'),
    (3, 'EUR', 'Euro'),
    (4, 'BGN', 'Bulgarian Lev'),
    (5, 'AUD', 'Australian Dollar'),
    (6, 'BRL', 'Brazilian Real'),
    (7, 'CAD', 'Canadian Dollar'),
    (8, 'CHF', 'Swiss Franc'),
    (9, 'CNY', 'Chinese Yuan'),
    (10, 'CZK', 'Czech Koruna'),
    (11, 'DKK', 'Danish Krone'),
    (12, 'GBP', 'British Pound'),
    (13, 'HKD', 'Hong Kong Dollar'),
    (14, 'HRK', 'Croatian Kuna'),
    (15, 'HUF', 'Hungarian Forint'),
    (16, 'IDR', 'Indonesian Rupiah'),
    (17, 'ILS', 'Israeli New Shekel'),
    (18, 'INR', 'Indian Rupee'),
    (19, 'ISK', 'Icelandic Krona'),
    (20, 'JPY', 'Japanese Yen'),
    (21, 'KRW', 'South Korean Won'),
    (22, 'MXN', 'Mexican Peso'),
    (23, 'MYR', 'Malaysian Ringgit'),
    (24, 'NOK', 'Norwegian Krone'),
    (25, 'NZD', 'New Zealand Dollar'),
    (26, 'PHP', 'Philippine Peso'),
    (27, 'PLN', 'Polish Zloty'),
    (28, 'RON', 'Romanian Leu'),
    (29, 'RUB', 'Russian Ruble'),
    (30, 'SEK', 'Swedish Krona'),
    (31, 'SGD', 'Singapore Dollar'),
    (32, 'THB', 'Thai Baht'),
    (33, 'TRY', 'Turkish Lira'),
    (34, 'ZAR', 'South African Rand')
ON CONFLICT (id) DO NOTHING;

SELECT setval('currencies_id_seq', (SELECT COALESCE(MAX(id), 0) FROM currencies));

-- ============================================================================
-- Account Types (6 rows)
-- ============================================================================
INSERT INTO account_types (id, type_name, is_credit) VALUES
    (1, 'cash', false),
    (2, 'regular_bank', false),
    (3, 'debit_card', false),
    (4, 'credit_card', true),
    (5, 'loan', true),
    (6, 'investment', false)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_types_id_seq', (SELECT COALESCE(MAX(id), 0) FROM account_types));

-- ============================================================================
-- Default Categories (21 rows)
-- Parent rows first, then children (due to self-referencing FK)
-- ============================================================================
INSERT INTO default_categories (id, name, parent_id, is_income, is_deleted) VALUES
    -- Expense parents
    (1, 'Life', NULL, false, false),
    (2, 'Food', NULL, false, false),
    (3, 'Automobile', NULL, false, false),
    (4, 'Transport', NULL, false, false),
    (5, 'Housing', NULL, false, false),
    (6, 'Health', NULL, false, false),
    (7, 'Education', NULL, false, false),
    (8, 'Entertainment', NULL, false, false),
    (9, 'Finances', NULL, false, false),
    (10, 'Other', NULL, false, false),
    -- Expense children
    (11, 'Parking', 3, false, false),
    (12, 'Fuel', 3, false, false),
    (13, 'Service', 3, false, false),
    (14, 'Taxi', 4, false, false),
    (15, 'Meat', 2, false, false),
    -- Income
    (16, 'Salary', NULL, true, false),
    (17, 'Deposit', NULL, true, false),
    (18, 'Present', NULL, true, false),
    (19, 'Rent', NULL, true, false),
    (20, 'Social', NULL, true, false),
    (21, 'Other', NULL, true, false)
ON CONFLICT (id) DO NOTHING;

SELECT setval('default_categories_id_seq', (SELECT COALESCE(MAX(id), 0) FROM default_categories));

-- ============================================================================
-- Languages (4 rows)
-- ============================================================================
INSERT INTO languages (id, name, code) VALUES
    (1, 'English', 'en'),
    (2, 'Русский', 'ru'),
    (3, 'Українська', 'uk'),
    (4, 'Български', 'bg'),
    (5, '简体中文', 'zh')
ON CONFLICT (id) DO NOTHING;

SELECT setval('languages_id_seq', (SELECT COALESCE(MAX(id), 0) FROM languages));

-- ============================================================================
-- Billing Periods (2 rows)
-- ============================================================================
INSERT INTO billing_periods (id, code, name, duration_days, sort_order) VALUES
    (1, 'MONTHLY', 'Monthly', 30, 1),
    (2, 'YEARLY', 'Yearly', 365, 2)
ON CONFLICT (id) DO NOTHING;

SELECT setval('billing_periods_id_seq', (SELECT COALESCE(MAX(id), 0) FROM billing_periods));

-- ============================================================================
-- Subscription Plans (4 rows)
-- Depends on: currencies (EUR = id 3), billing_periods
-- ============================================================================
INSERT INTO subscription_plans (id, name, translation_key, plan_type, billing_period_id, currency_id, price, is_active, is_featured, sort_order, description) VALUES
    (1, 'Free Plan', 'plans.free', 'FREE', NULL, 3, 0.00, true, false, 0, 'Free plan with basic features'),
    (2, 'Trial Plan', 'plans.trial', 'TRIAL', NULL, 3, 0.00, true, false, -1, '45-day trial with full access to all features'),
    (3, 'Premium Monthly', 'plans.premiumMonthly', 'PREMIUM', 1, 3, 5.00, true, false, 1, 'Monthly premium subscription with full access to all features'),
    (4, 'Premium Yearly', 'plans.premiumYearly', 'PREMIUM', 2, 3, 50.00, true, true, 2, 'Yearly premium subscription - save 2 months! (17% discount)')
ON CONFLICT (id) DO NOTHING;

SELECT setval('subscription_plans_id_seq', (SELECT COALESCE(MAX(id), 0) FROM subscription_plans));

-- ============================================================================
-- Goose migration history
-- Mark all existing migrations as applied so future `goose up` works correctly
-- ============================================================================
INSERT INTO goose_db_version (version_id, is_applied) VALUES
    (0, true),
    (1, true),
    (2, true)
ON CONFLICT DO NOTHING;

COMMIT;
