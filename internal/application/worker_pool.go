package application

import (
	"context"
	"log/slog"
	"sync"

	"notification-processor/internal/domain"
)

type usecase interface {
	Execute(ctx context.Context, event domain.Event) error
}

// job — единица работы: событие + callback для коммита offset после обработки.
// Callback определён как функция чтобы WorkerPool не зависел от Kafka —
// он не знает что такое offset, просто вызывает функцию по завершению.
type job struct {
	Event        domain.Event
	CommitOffset func() error
}

func NewJob(Event domain.Event, CommitOffset func() error) *job {
	return &job{Event: Event, CommitOffset: CommitOffset}
}

// WorkerPool — пул воркеров фиксированного размера.
//
// Consumer читает из Kafka и отдаёт события в пул.
// Пул обрабатывает их конкурентно через ProcessEventUseCase.
//
// Фикс размер = фикс кол-во горутин, чтобы не было утечек памяти при любом количестве ошибок и retry.
//
//	Consumer -> jobs channel -> [w1][w2]...[wN] -> ProcessEventUseCase.Execute()
type WorkerPool struct {
	workerCount int
	jobs        chan job
	usecase     usecase
	wg          sync.WaitGroup
}

func NewWorkerPool(workerCount int, usecase usecase) *WorkerPool {
	return &WorkerPool{
		// Буфер = workerCount * 2: consumer может читать вперёд пока воркеры заняты текущими задачами.
		// Не слишком большой — не хотим держать много неподтверждённых офсетов.
		workerCount: workerCount,
		jobs:        make(chan job, workerCount*2),
		usecase:     usecase,
	}
}

// Start запускает N воркеров и блокируется до их завершения.
// Запускать в отдельной горутине: go pool.Start(ctx)
func (p *WorkerPool) Start(ctx context.Context) {
	slog.Info("worker pool starting", "workers", p.workerCount)

	for i := range p.workerCount {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			p.runWorker(ctx, id)
		}(i + 1)
	}

	p.wg.Wait()
	slog.Info("worker pool stopped")
}

// Submit передаёт событие на обработку.
// Блокируется если все воркеры заняты и канал полон consumer притормозит чтение.
// Возвращает false если ctx отменён (graceful shutdown).
func (p *WorkerPool) Submit(ctx context.Context, job job) bool {
	select {
	case p.jobs <- job:
		return true
	case <-ctx.Done():
		return false
	}
}

// Stop закрывает канал задач.
// Воркеры обработают всё что уже попало в канал и завершатся.
// Вызывать после того как consumer прекратил читать.
func (p *WorkerPool) Stop() {
	close(p.jobs)
}

// Wait блокируется до завершения всех воркеров.
// Использовать в связке со Stop для graceful shutdown:
//
//	pool.Stop()
//	pool.Wait()
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// runWorker — основной цикл воркера.
func (p *WorkerPool) runWorker(ctx context.Context, id int) {
	log := slog.With("worker_id", id)
	log.Debug("worker started")

	for job := range p.jobs {
		p.processJob(ctx, log, job)
	}

	log.Debug("worker stopped")
}

func (p *WorkerPool) processJob(ctx context.Context, log *slog.Logger, job job) {
	log = log.With("event_id", job.Event.EventID, "event_type", job.Event.EventType)

	err := p.usecase.Execute(ctx, job.Event)

	switch err {
	case nil, ErrAlreadyProcessed, ErrUnknownEventType:
		// Успех или задекларированная ошибка - коммитим offset
		if commitErr := job.CommitOffset(); commitErr != nil {
			log.Error("failed to commit offset", "error", commitErr)
		}

	default:
		// Инфраструктурная ошибка не коммитим offset, Kafka переотправит сообщение после рестарта.
		log.Error("failed to process event, offset not committed", "error", err)
	}
}
