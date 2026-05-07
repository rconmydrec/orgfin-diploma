-- +goose Up
INSERT INTO languages (id, name, code) VALUES
    (4, 'Български', 'bg')
ON CONFLICT (id) DO NOTHING;

SELECT setval('languages_id_seq', (SELECT COALESCE(MAX(id), 0) FROM languages));

-- +goose Down
DELETE FROM languages WHERE code = 'bg';
