-- +goose Up
-- +goose StatementBegin
-- Carry album and album-artist metadata through the pipeline so the worker can
-- resolve a clean primary artist (album-artist preferred over a possibly
-- multi-valued track artist) and pass q_album to the matcher. Plain ADD COLUMN
-- needs no table rebuild here (no CHECK/constraint change). Existing rows
-- default to '' and are backfilled on the next library scan upsert.
ALTER TABLE scan_results ADD COLUMN album TEXT NOT NULL DEFAULT '';
ALTER TABLE scan_results ADD COLUMN album_artist TEXT NOT NULL DEFAULT '';
ALTER TABLE work_queue ADD COLUMN album TEXT NOT NULL DEFAULT '';
ALTER TABLE work_queue ADD COLUMN album_artist TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE work_queue DROP COLUMN album_artist;
ALTER TABLE work_queue DROP COLUMN album;
ALTER TABLE scan_results DROP COLUMN album_artist;
ALTER TABLE scan_results DROP COLUMN album;
-- +goose StatementEnd
