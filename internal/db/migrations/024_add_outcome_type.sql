-- +goose Up
-- +goose StatementBegin
-- outcome_type records what was actually written when a work_queue row
-- completed (issue #379): 'synced' (a .lrc synced-lyrics file), 'unsynced' (a
-- .txt unsynced-lyrics file), 'instrumental' (a .txt instrumental marker), or
-- NULL (nothing written -- guard-rejected -- or a legacy row predating this
-- column). Before this, the dashboard/reports classified a completed row by the
-- filename extension stored in output_paths, but output_paths is marshaled once
-- at enqueue time (always the planned .lrc) and never updated at completion, so
-- every done row read as 'synced' and unsynced/instrumental outcomes were
-- invisible. The worker now stamps the true outcome via SetOutcomeType before
-- Complete(), and reports read this column instead of parsing output_paths.
ALTER TABLE work_queue ADD COLUMN outcome_type TEXT;
-- +goose StatementEnd
-- +goose StatementBegin
-- Reliable partial backfill: rows the audio detector already confirmed as
-- instrumental (instrumental_result = 1) are known-good, so recover them so they
-- classify correctly without waiting for reprocessing. Remaining legacy rows
-- stay NULL ('unknown' in reports) because their true outcome is not reliably
-- reconstructable from the DB -- output_paths holds the stale enqueue-time .lrc,
-- not what was written -- and re-reading every output file from disk is out of
-- scope and unreliable.
UPDATE work_queue SET outcome_type = 'instrumental'
    WHERE instrumental_result = 1 AND status = 'done';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE work_queue DROP COLUMN outcome_type;
-- +goose StatementEnd
