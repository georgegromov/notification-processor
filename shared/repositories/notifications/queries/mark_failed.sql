UPDATE notifications
SET status = 'failed',
    attempts = $2,
    locked_at = NULL,
    last_error = $3,
    updated_at = now()
WHERE id = $1 AND status = 'processing';