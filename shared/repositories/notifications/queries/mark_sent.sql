UPDATE notifications
SET status = 'sent',
    locked_at = NULL,
    last_error = NULL,
    updated_at = now()
WHERE id = $1 AND status = 'processing';