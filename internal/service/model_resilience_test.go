package service

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestModelGuardRetriesTransientHTTPFailures(t *testing.T) {
	guard := newModelGuard("test-chat")
	guard.cfg.MaxRetries = 2
	guard.cfg.RetryBackoff = time.Millisecond
	var calls atomic.Int32

	err := guard.run(context.Background(), func(context.Context) error {
		if calls.Add(1) < 3 {
			return &modelHTTPError{status: http.StatusBadGateway}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
	if got := guard.health().State; got != "healthy" {
		t.Fatalf("state = %q, want healthy", got)
	}
}

func TestModelGuardDoesNotRetryClientErrors(t *testing.T) {
	guard := newModelGuard("test-chat")
	guard.cfg.RetryBackoff = time.Millisecond
	var calls atomic.Int32

	err := guard.run(context.Background(), func(context.Context) error {
		calls.Add(1)
		return &modelHTTPError{status: http.StatusUnauthorized}
	})
	if err == nil || calls.Load() != 1 {
		t.Fatalf("error = %v, calls = %d, want one non-retried call", err, calls.Load())
	}
}

func TestModelGuardOpensAndHalfOpensCircuit(t *testing.T) {
	guard := newModelGuard("test-chat")
	guard.cfg.MaxRetries = 0
	guard.cfg.FailureThreshold = 2
	guard.cfg.OpenDuration = 20 * time.Millisecond
	var calls atomic.Int32

	fail := func(context.Context) error {
		calls.Add(1)
		return &modelHTTPError{status: http.StatusBadGateway}
	}
	_ = guard.run(context.Background(), fail)
	_ = guard.run(context.Background(), fail)
	if got := guard.health().State; got != "open" {
		t.Fatalf("state = %q, want open", got)
	}
	if err := guard.run(context.Background(), fail); !errors.Is(err, ErrModelCircuitOpen) {
		t.Fatalf("closed call error = %v, want ErrModelCircuitOpen", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls while open = %d, want 2", got)
	}

	time.Sleep(25 * time.Millisecond)
	if err := guard.run(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("half-open probe failed: %v", err)
	}
	if got := guard.health().State; got != "healthy" {
		t.Fatalf("state after probe = %q, want healthy", got)
	}
}

func TestModelGuardStreamingDoesNotRetryAfterToken(t *testing.T) {
	guard := newModelGuard("test-stream")
	guard.cfg.MaxRetries = 2
	guard.cfg.RetryBackoff = time.Millisecond
	var calls atomic.Int32

	answer, err := guard.runStreaming(context.Background(), func(context.Context) (string, bool, error) {
		calls.Add(1)
		return "partial", true, &modelHTTPError{status: http.StatusBadGateway}
	})
	if answer != "partial" || err == nil || calls.Load() != 1 {
		t.Fatalf("answer=%q err=%v calls=%d, want partial single attempt", answer, err, calls.Load())
	}
}

func TestModelGuardEnforcesPerAttemptTimeout(t *testing.T) {
	guard := newModelGuard("test-timeout")
	guard.cfg.Timeout = 15 * time.Millisecond
	guard.cfg.MaxRetries = 0

	started := time.Now()
	err := guard.run(context.Background(), func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("timeout took %s, want under 500ms", elapsed)
	}
}

func TestModelGuardAllowsOnlyOneHalfOpenProbe(t *testing.T) {
	guard := newModelGuard("test-half-open")
	guard.cfg.MaxRetries = 0
	guard.cfg.FailureThreshold = 1
	guard.cfg.OpenDuration = 20 * time.Millisecond

	_ = guard.run(context.Background(), func(context.Context) error {
		return &modelHTTPError{status: http.StatusBadGateway}
	})
	time.Sleep(25 * time.Millisecond)

	probeStarted := make(chan struct{})
	releaseProbe := make(chan struct{})
	probeResult := make(chan error, 1)
	go func() {
		probeResult <- guard.run(context.Background(), func(context.Context) error {
			close(probeStarted)
			<-releaseProbe
			return nil
		})
	}()
	<-probeStarted

	secondErr := guard.run(context.Background(), func(context.Context) error {
		t.Fatal("second half-open probe reached provider")
		return nil
	})
	if !errors.Is(secondErr, ErrModelCircuitOpen) {
		t.Fatalf("second probe error = %v, want ErrModelCircuitOpen", secondErr)
	}
	close(releaseProbe)
	if err := <-probeResult; err != nil {
		t.Fatalf("first half-open probe failed: %v", err)
	}
}

func TestModelGuardCancellationDoesNotOpenCircuit(t *testing.T) {
	guard := newModelGuard("test-canceled")
	guard.cfg.MaxRetries = 0
	guard.cfg.FailureThreshold = 1

	for range 3 {
		if err := guard.run(context.Background(), func(context.Context) error {
			return context.Canceled
		}); !errors.Is(err, context.Canceled) {
			t.Fatalf("run error = %v, want context canceled", err)
		}
	}

	health := guard.health()
	if health.State != "healthy" || health.ConsecutiveFailures != 0 {
		t.Fatalf("health after cancellations = %+v, want healthy with zero failures", health)
	}
}

func TestModelResultClassifiesCancellation(t *testing.T) {
	if got := modelResult(context.Canceled); got != "canceled" {
		t.Fatalf("modelResult(context.Canceled) = %q, want canceled", got)
	}
}

func TestModelGuardStreamingCancellationDoesNotOpenCircuit(t *testing.T) {
	guard := newModelGuard("test-stream-canceled")
	guard.cfg.MaxRetries = 0
	guard.cfg.FailureThreshold = 1

	_, err := guard.runStreaming(context.Background(), func(context.Context) (string, bool, error) {
		return "", false, context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runStreaming error = %v, want context canceled", err)
	}

	health := guard.health()
	if health.State != "healthy" || health.ConsecutiveFailures != 0 {
		t.Fatalf("health after cancellation = %+v, want healthy with zero failures", health)
	}
}

func TestModelGuardCancellationReleasesHalfOpenProbe(t *testing.T) {
	guard := newModelGuard("test-half-open-canceled")
	guard.cfg.MaxRetries = 0
	guard.cfg.FailureThreshold = 1
	guard.cfg.OpenDuration = 10 * time.Millisecond

	_ = guard.run(context.Background(), func(context.Context) error {
		return &modelHTTPError{status: http.StatusBadGateway}
	})
	time.Sleep(15 * time.Millisecond)

	if err := guard.run(context.Background(), func(context.Context) error {
		return context.Canceled
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("half-open probe error = %v, want context canceled", err)
	}
	if err := guard.run(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("probe after cancellation failed: %v", err)
	}
}
