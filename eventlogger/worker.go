package eventlogger

import (
	"context"
	"log/slog"
	"sync"
)

type Worker struct {
	eventCh chan Event
	logger  EventLogger
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorker(logger EventLogger, bufferSize int) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		eventCh: make(chan Event, bufferSize),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (w *Worker) Start() {
	w.wg.Go(func() {
		for {
			select {
			case <-w.ctx.Done():
				slog.Info("draining events before shutdown", "remaining_events", len(w.eventCh))
				for len(w.eventCh) > 0 {
					event := <-w.eventCh
					if err := w.logger.Save(context.Background(), event); err != nil {
						slog.Error("failed to save event during shutdown", "error", err, "event_type", event.Type)
					}
				}
				return
			case event := <-w.eventCh:
				if err := w.logger.Save(w.ctx, event); err != nil {
					slog.Error("failed to save event", "error", err, "event_type", event.Type)
				}
			}
		}
	})
}

func (w *Worker) Log(event Event) {
	select {
	case w.eventCh <- event:
		// Event sent successfully
	default:
		// Channel is full, log the error
		slog.Warn("event channel full, dropping event", "event_type", event.Type)
	}
}

func (w *Worker) EventChannel() chan<- Event {
	return w.eventCh
}

func (w *Worker) Shutdown() {
	w.cancel()
	w.wg.Wait()
	close(w.eventCh)
}
