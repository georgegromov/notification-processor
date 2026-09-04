INSERT INTO processed_events (event_id, kafka_topic, kafka_partition, kafka_offset)
VALUES ($1, $2, $3, $4)
ON CONFLICT (event_id) DO NOTHING;