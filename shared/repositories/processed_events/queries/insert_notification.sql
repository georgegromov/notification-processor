INSERT INTO notifications (event_id, user_id, channel, status)
VALUES ($1, $2, $3, 'pending')
ON CONFLICT (event_id, channel) DO NOTHING;