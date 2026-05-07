-- +goose Up
INSERT INTO languages (id, name, code) VALUES
    (5, '简体中文', 'zh')
ON CONFLICT (id) DO NOTHING;

SELECT setval('languages_id_seq', (SELECT COALESCE(MAX(id), 0) FROM languages));

-- +goose Down
DELETE FROM languages WHERE code = 'zh';
