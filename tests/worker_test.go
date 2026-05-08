package tests

// Tests unitarios del worker pool.
//
// No requieren base de datos ni Docker — todo es en memoria con goroutines reales.
// Esto demuestra que el pool es testeable de forma aislada porque su única
// dependencia externa es la función Processor, que en tests es un simple closure.
//
// Para ejecutar:
//
//	go test -v -run TestWorker ./tests/

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oshy/score-gorad/internal/worker"
)

// TestWorkerProcessesEvents verifica que todos los eventos enviados
// son procesados por el pool.
func TestWorkerProcessesEvents(t *testing.T) {
	const numEvents = 50
	var processed atomic.Int64

	pool := worker.New(128)
	pool.Start(4, func(e worker.ScoreEvent) error {
		processed.Add(1)
		return nil
	})

	for i := range numEvents {
		event := worker.ScoreEvent{
			GameID:   "game-1",
			PlayerID: "player-" + string(rune('A'+i%26)),
			Points:   int64(i * 100),
		}
		if err := pool.Submit(event); err != nil {
			t.Fatalf("unexpected submit error: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	if got := processed.Load(); got != numEvents {
		t.Errorf("expected %d processed events, got %d", numEvents, got)
	}
}

// TestWorkerBackpressure verifica que Submit devuelve ErrPoolFull
// cuando el canal está saturado, sin bloquear al caller.
func TestWorkerBackpressure(t *testing.T) {
	// Pool con buffer muy pequeño y processor lento para saturarlo fácilmente.
	pool := worker.New(2)
	pool.Start(1, func(e worker.ScoreEvent) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	var fullCount int
	for range 20 {
		err := pool.Submit(worker.ScoreEvent{GameID: "g", PlayerID: "p", Points: 1})
		if errors.Is(err, worker.ErrPoolFull) {
			fullCount++
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = pool.Shutdown(ctx)

	if fullCount == 0 {
		t.Error("expected at least one ErrPoolFull with a small buffer, got none")
	}
}

// TestWorkerShutdownDrainsInFlight verifica que Shutdown espera a que
// los jobs en vuelo terminen antes de retornar.
func TestWorkerShutdownDrainsInFlight(t *testing.T) {
	var wg sync.WaitGroup
	var processed atomic.Int64
	const numEvents = 10

	pool := worker.New(128)
	pool.Start(2, func(e worker.ScoreEvent) error {
		time.Sleep(20 * time.Millisecond) // simula trabajo real
		processed.Add(1)
		wg.Done()
		return nil
	})

	wg.Add(numEvents)
	for range numEvents {
		_ = pool.Submit(worker.ScoreEvent{GameID: "g", PlayerID: "p", Points: 100})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}

	// Esperar que todos los WaitGroup done lleguen (ya deberían si Shutdown drenó)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Errorf("not all events were processed before shutdown returned; processed=%d", processed.Load())
	}
}

// TestWorkerShutdownRespectsContextDeadline verifica que Shutdown respeta
// el deadline del contexto y devuelve ctx.Err() si los workers no terminan a tiempo.
func TestWorkerShutdownRespectsContextDeadline(t *testing.T) {
	pool := worker.New(128)
	pool.Start(1, func(e worker.ScoreEvent) error {
		time.Sleep(500 * time.Millisecond) // más lento que el deadline
		return nil
	})

	_ = pool.Submit(worker.ScoreEvent{GameID: "g", PlayerID: "p", Points: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := pool.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

// TestWorkerRetryOnError verifica que el pool reintenta un evento
// fallido el número configurado de veces antes de darlo por perdido.
func TestWorkerRetryOnError(t *testing.T) {
	var attempts atomic.Int64
	expectedAttempts := int64(3) // maxRetries=2 → 3 intentos totales

	pool := worker.New(128,
		worker.WithRetry(2, time.Millisecond),
		worker.WithErrorHandler(func(e worker.ScoreEvent, err error, total int) {
			// El ErrorHandler recibe el número total de intentos realizados.
		}),
	)
	pool.Start(1, func(e worker.ScoreEvent) error {
		attempts.Add(1)
		return errors.New("simulated failure")
	})

	_ = pool.Submit(worker.ScoreEvent{GameID: "g", PlayerID: "p", Points: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = pool.Shutdown(ctx)

	if got := attempts.Load(); got != expectedAttempts {
		t.Errorf("expected %d attempts (1 initial + 2 retries), got %d", expectedAttempts, got)
	}
}

// TestWorkerErrorHandlerCalled verifica que el ErrorHandler se invoca
// cuando un evento falla todos sus reintentos.
func TestWorkerErrorHandlerCalled(t *testing.T) {
	var handlerCalled atomic.Bool

	pool := worker.New(128,
		worker.WithRetry(1, time.Millisecond),
		worker.WithErrorHandler(func(e worker.ScoreEvent, err error, total int) {
			handlerCalled.Store(true)
		}),
	)
	pool.Start(1, func(e worker.ScoreEvent) error {
		return errors.New("always fails")
	})

	_ = pool.Submit(worker.ScoreEvent{GameID: "g", PlayerID: "p", Points: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = pool.Shutdown(ctx)

	if !handlerCalled.Load() {
		t.Error("expected ErrorHandler to be called after all retries exhausted")
	}
}
