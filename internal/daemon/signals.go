package daemon

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SignalHandler handles OS signals for graceful shutdown.
type SignalHandler struct {
	signals chan os.Signal
	done    chan struct{}
}

// NewSignalHandler creates a new signal handler.
func NewSignalHandler() *SignalHandler {
	return &SignalHandler{
		signals: make(chan os.Signal, 1),
		done:    make(chan struct{}),
	}
}

// Setup registers signal handlers.
func (h *SignalHandler) Setup() {
	signal.Notify(h.signals,
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // Termination request
		syscall.SIGHUP,  // Terminal hangup
	)
}

// Wait blocks until a shutdown signal is received or context is cancelled.
func (h *SignalHandler) Wait(ctx context.Context) os.Signal {
	select {
	case sig := <-h.signals:
		return sig
	case <-ctx.Done():
		return nil
	case <-h.done:
		return nil
	}
}

// Stop stops waiting for signals.
func (h *SignalHandler) Stop() {
	signal.Stop(h.signals)
	close(h.done)
}

// Cleanup performs cleanup after signal handling.
func (h *SignalHandler) Cleanup() {
	signal.Stop(h.signals)
}
