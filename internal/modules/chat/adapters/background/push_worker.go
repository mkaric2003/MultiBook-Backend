package background

import (
	"context"
	"log/slog"
	"time"
)

const (
	pushPollInterval = time.Second
	pushJobTimeout   = 20 * time.Second
)

type Dispatcher interface {
	ProcessNext(context.Context) (bool, error)
}

type PushWorker struct {
	dispatcher Dispatcher
	logger     *slog.Logger
}

func NewPushWorker(dispatcher Dispatcher, logger *slog.Logger) *PushWorker {
	return &PushWorker{dispatcher: dispatcher, logger: logger}
}

func (w *PushWorker) Run(ctx context.Context) {
	go w.run(ctx)
}

func (w *PushWorker) run(ctx context.Context) {
	w.drain(ctx)
	ticker := time.NewTicker(pushPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.drain(ctx)
		}
	}
}

func (w *PushWorker) drain(ctx context.Context) {
	for ctx.Err() == nil {
		jobContext, cancel := context.WithTimeout(ctx, pushJobTimeout)
		processed, err := w.dispatcher.ProcessNext(jobContext)
		cancel()
		if err != nil {
			w.logger.Warn("chat push delivery failed", "error", err)
			return
		}
		if !processed {
			return
		}
	}
}
