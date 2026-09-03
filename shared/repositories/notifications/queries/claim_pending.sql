UPDATE notifications
SET status = 'processing',
    locked_at = now(),
    updated_at = now()
WHERE id IN (
    SELECT id
    FROM notifications
    WHERE channel = $1
      AND status = 'pending'
      AND next_retry_at <= now()
    ORDER BY next_retry_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
RETURNING id, event_id, user_id, channel, status, attempts, next_retry_at, locked_at, last_error, created_at, updated_at;