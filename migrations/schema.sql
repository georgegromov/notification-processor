-- =============================================================================
-- Schema: notification-processor
-- Применяется автоматически при старте сервиса
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Таблица 1: processed_events
--
-- Назначение: дедупликация + exactly-once семантика
--
-- Как работает:
-- 		При получении сообщения из Kafka делаем:
--			INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING
--   	Если вернулось 0 rows — событие уже обрабатывалось, пропускаем.
--   	Это защищает от повторной обработки при перезапуске после краша.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     UUID        PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- -----------------------------------------------------------------------------
-- Таблица 2: notifications
--
-- Назначение: хранение состояния каждого уведомления
--
-- Одно событие Kafka -> несколько уведомлений (по числу каналов):
--   order_created -> 2 строки: email + push
--   payment_received -> 1 строка:  email
--   order_shipped -> 2 строки: push + sms
--
-- Жизненный цикл статуса:
-- 	pending -> delivered (успех)
--	pending -> failed_temp (исчерпаны попытки, была временная ошибка)
--  pending -> failed_perm (перманентная ошибка, повтор бессмысленен)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id          BIGSERIAL   PRIMARY KEY,
    event_id    UUID        NOT NULL,
    user_id     BIGINT      NOT NULL,
    channel     TEXT        NOT NULL, -- 'email' | 'push' | 'sms'
    status      TEXT        NOT NULL DEFAULT 'pending', -- 'pending' | 'delivered' | 'failed_temp' | 'failed_perm'
    attempts    INT         NOT NULL DEFAULT 0,
    last_error  TEXT, -- последнее сообщение об ошибке (для дебага)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

		CONSTRAINT fk_event_id FOREIGN KEY (event_id) 
			REFERENCES processed_events(event_id)
);

-- Индекс для быстрого поиска уведомлений по событию (нужен при повторных попытках)
CREATE INDEX IF NOT EXISTS idx_notifications_event_id ON notifications (event_id);

-- Составной уникальный индекс: одно уведомление на событие + канал
-- Защита от двойной вставки на уровне БД
CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_event_channel ON notifications (event_id, channel);