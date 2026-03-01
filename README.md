# Notification Processor

Сервис читает события пользовательских действий из Kafka, формирует персонализированные уведомления
по каналам (email, push, sms) и гарантирует их доставку. Состояние хранится в PostgreSQL, доставка
эмулируется с имитацией сбоев.

## Возможности

- **Идемпотентность** — повторное сообщение с тем же `event_id` не создаёт дубликат.
- **Exactly-once** — сообщение не теряется при падении между записью в БД и коммитом offset в Kafka.
- **Poison pill** — битый JSON логируется, offset коммитится, обработка остальных сообщений
  продолжается.
- **Без утечек** — фиксированный пул воркеров, при постоянных временных ошибках память не растёт.
- **Graceful shutdown** — по SIGTERM/SIGINT текущие сообщения дообрабатываются, затем выход.

## Маршрутизация

| event_type       | Каналы      |
| ---------------- | ----------- |
| order_created    | email, push |
| payment_received | email       |
| order_shipped    | push, sms   |

## Эмуляция доставки

Эмулятор имитирует внешний сервис доставки:

- ~89% — успех с первой или после ретраев попытки.
- ~10% — временная ошибка (TempError): ретраи с экспоненциальной задержкой.
- ~1% — перманентная ошибка (PermError): ретраи до `RETRY_MAX_ATTEMPTS`, затем прекращение.

После исчерпания попыток статус в БД: `failed_temp` или `failed_perm`.

## Формат сообщения Kafka

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 12345,
  "event_type": "order_created",
  "timestamp": "2026-03-01T15:30:00Z",
  "payload": { "order_id": 99, "amount": 1500 }
}
```

Поля `event_id` (UUID), `user_id`, `event_type`, `timestamp` обязательны. `payload` — произвольный
JSON, может отсутствовать.

## Требования

- Go 1.24+
- Docker
- PostgreSQL 16, Kafka (Confluent 7.x или аналог)

## Быстрый старт

Поднять инфраструктуру и сервис:

```bash
docker-compose up -d --build
```

Сервис подключится к Kafka и PostgreSQL, применит схему из `migrations/schema.sql` и начнёт
потреблять топик `user-events`. Отправить тестовые события:

```bash
docker-compose run --rm producer
```

## Структура проекта

```
.
├── cmd/
│   ├── notification-processor/   # основной сервис
│   └── event-producer/          # утилита для тестовых событий
├── internal/
│   ├── application/            # use case, пул воркеров
│   ├── config/                  # загрузка конфига
│   ├── domain/                  # Event, Notification, роутинг, ошибки
│   ├── repository/              # PostgreSQL (processed_events, notifications)
│   └── transport/
│       ├── consumer/            # Kafka consumer, parseMessage
│       └── emulator/             # эмулятор доставки с retry/backoff
├── pkg/db/                       # пул БД, применение схемы
├── migrations/
│   └── schema.sql               # таблицы и индексы
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── go.mod
```

## Тесты

```bash
go test ./...
```

Тесты покрывают домен, сервис обработки события, пул воркеров, парсинг сообщений consumer и retry/backoff эмулятора.
