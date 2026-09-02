-- Inbox: processed events (deduplication by event_id)
CREATE TABLE IF NOT EXISTS processed_events (
	event_id UUID PRIMARY KEY,
	topic TEXT NOT NULL,
	partition INT NOT NULL,
	offset BIGINT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Outbox: notification delivery queue
CREATE TABLE IF NOT EXISTS notifications (
	id BIGSERIAL PRIMARY KEY,
	event_id UUID NOT NULL,
	user_id BIGINT NOT NULL,
	channel TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	attempts INT NOT NULL DEFAULT 0,
	next_retry_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	locked_at TIMESTAMPTZ,
	last_error TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
);

ALTER TABLE only notifications
	ADD CONSTRAINT notifications_event_channel_unique UNIQUE (event_id, channel),
	ADD CONSTRAINT notifications_status_check CHECK (status IN ('pending', 'processing', 'sent', 'failed')),
	ADD CONSTRAINT notifications_channel_check CHECK (channel IN ('email', 'push', 'sms'));

-- Dispatcher: claim pending tasks by channel
CREATE INDEX IF NOT EXISTS idx_notifications_pending_dispatch
	ON notifications (channel, next_retry_at)
	WHERE status = 'pending';

-- Dispatcher: reclaim stale processing tasks (lease expired)
CREATE INDEX IF NOT EXISTS idx_notifications_stale_processing
	ON notifications (locked_at)
	WHERE status = 'processing';
