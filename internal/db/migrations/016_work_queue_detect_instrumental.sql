-- +goose Up
-- +goose StatementBegin
-- Per-item instrumental-detection decision, resolved and stamped at enqueue time
-- (mirrors providers_version). NULL = no decision stamped; the worker falls back
-- to the global config default (covers all pre-existing rows, preserving current
-- behavior). 0 = detection off for this item, 1 = on. Plain ADD COLUMN needs no
-- table rebuild (no CHECK/constraint change); existing rows default to NULL.
ALTER TABLE work_queue ADD COLUMN detect_instrumental INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE work_queue DROP COLUMN detect_instrumental;
-- +goose StatementEnd
