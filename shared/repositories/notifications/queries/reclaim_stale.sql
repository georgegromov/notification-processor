UPDATE notifications
SET status = 'pending',
    locked_at = NULL,
    updated_at = now()
WHERE status = 'processing'
  AND locked_at < now() - ($1::text || ' seconds')::interval;