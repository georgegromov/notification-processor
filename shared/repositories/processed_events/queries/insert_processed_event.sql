INSERT INTO processed_events (event_id, topic, partition, offset)
VALUES ($1, $2, $3, $4)
ON CONFLICT (event_id) DO NOTHING;