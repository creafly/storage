DROP INDEX IF EXISTS idx_outbox_events_created_at;
DROP INDEX IF EXISTS idx_outbox_events_status_retry;
DROP TABLE IF EXISTS outbox_events;
