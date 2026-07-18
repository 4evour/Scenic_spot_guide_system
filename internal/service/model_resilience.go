package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/scenic-guide/internal/pkg"
)

var ErrModelCircuitOpen = errors.New("model provider circuit is open")

type modelGuardConfig struct {
	Timeout          time.Duration
	MaxRetries       int
	FailureThreshold int
	OpenDuration     time.Duration
	RetryBackoff     time.Duration
}

var defaultModelGuardConfig = modelGuardConfig{
	Timeout:          60 * time.Second,
	MaxRetries:       2,
	FailureThreshold: 3,
	OpenDuration:     30 * time.Second,
	RetryBackoff:     100 * time.Millisecond,
}

type ModelProviderHealth struct {
	Provider            string    `json:"provider"`
	State               string    `json:"state"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	OpenUntil           time.Time `json:"open_until,omitempty"`
}

type modelGuard struct {
	mu sync.Mutex

	provider            string
	cfg                 modelGuardConfig
	consecutiveFailures int
	openUntil           time.Time
	probeInFlight       bool
}

func newModelGuard(provider string) *modelGuard {
	return &modelGuard{provider: provider, cfg: defaultModelGuardConfig}
}

func (g *modelGuard) run(ctx context.Context, fn func(context.Context) error) error {
	if err := g.acquire(); err != nil {
		pkg.RecordModelCall(g.provider, "circuit_open", 0)
		pkg.RecordModelCircuitOpen(g.provider)
		return err
	}

	started := time.Now()
	var lastErr error
	for attempt := 0; attempt <= g.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			g.recordFailure(err)
			pkg.RecordModelCall(g.provider, modelResult(err), time.Since(started))
			return err
		}

		attemptCtx, cancel := context.WithTimeout(ctx, g.cfg.Timeout)
		err := fn(attemptCtx)
		cancel()
		if err == nil {
			g.recordSuccess()
			pkg.RecordModelCall(g.provider, "success", time.Since(started))
			return nil
		}
		lastErr = err
		if !isRetryableModelError(err) || attempt == g.cfg.MaxRetries {
			g.recordFailure(err)
			pkg.RecordModelCall(g.provider, modelResult(err), time.Since(started))
			return err
		}

		pkg.RecordModelRetry(g.provider)
		if err := waitModelRetry(ctx, g.cfg.RetryBackoff, attempt); err != nil {
			g.recordFailure(err)
			pkg.RecordModelCall(g.provider, modelResult(err), time.Since(started))
			return err
		}
	}

	g.recordFailure(lastErr)
	pkg.RecordModelCall(g.provider, modelResult(lastErr), time.Since(started))
	return lastErr
}

// runStreaming retries only before the provider sends the first token. Retrying
// after tokens were emitted would duplicate visible content for the caller.
func (g *modelGuard) runStreaming(ctx context.Context, fn func(context.Context) (string, bool, error)) (string, error) {
	if err := g.acquire(); err != nil {
		pkg.RecordModelCall(g.provider, "circuit_open", 0)
		pkg.RecordModelCircuitOpen(g.provider)
		return "", err
	}

	started := time.Now()
	var lastErr error
	for attempt := 0; attempt <= g.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			g.recordFailure(err)
			pkg.RecordModelCall(g.provider, modelResult(err), time.Since(started))
			return "", err
		}

		attemptCtx, cancel := context.WithTimeout(ctx, g.cfg.Timeout)
		answer, emitted, err := fn(attemptCtx)
		cancel()
		if err == nil {
			g.recordSuccess()
			pkg.RecordModelCall(g.provider, "success", time.Since(started))
			return answer, nil
		}
		lastErr = err
		if emitted || !isRetryableModelError(err) || attempt == g.cfg.MaxRetries {
			g.recordFailure(err)
			pkg.RecordModelCall(g.provider, modelResult(err), time.Since(started))
			return answer, err
		}

		pkg.RecordModelRetry(g.provider)
		if err := waitModelRetry(ctx, g.cfg.RetryBackoff, attempt); err != nil {
			g.recordFailure(err)
			pkg.RecordModelCall(g.provider, modelResult(err), time.Since(started))
			return "", err
		}
	}

	g.recordFailure(lastErr)
	pkg.RecordModelCall(g.provider, modelResult(lastErr), time.Since(started))
	return "", lastErr
}

func (g *modelGuard) acquire() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	if now.Before(g.openUntil) {
		return fmt.Errorf("%w: %s", ErrModelCircuitOpen, g.provider)
	}
	if !g.openUntil.IsZero() {
		if g.probeInFlight {
			return fmt.Errorf("%w: %s", ErrModelCircuitOpen, g.provider)
		}
		g.probeInFlight = true
	}
	return nil
}

func (g *modelGuard) recordSuccess() {
	g.mu.Lock()
	g.consecutiveFailures = 0
	g.openUntil = time.Time{}
	g.probeInFlight = false
	g.mu.Unlock()
}

func (g *modelGuard) recordFailure(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err == nil {
		return
	}
	g.probeInFlight = false
	if errors.Is(err, context.Canceled) {
		return
	}
	g.consecutiveFailures++
	if g.consecutiveFailures >= g.cfg.FailureThreshold {
		g.openUntil = time.Now().Add(g.cfg.OpenDuration)
	}
}

func (g *modelGuard) health() ModelProviderHealth {
	g.mu.Lock()
	defer g.mu.Unlock()

	state := "healthy"
	if time.Now().Before(g.openUntil) {
		state = "open"
	} else if !g.openUntil.IsZero() {
		state = "half_open"
	}
	return ModelProviderHealth{
		Provider:            g.provider,
		State:               state,
		ConsecutiveFailures: g.consecutiveFailures,
		OpenUntil:           g.openUntil,
	}
}

func waitModelRetry(ctx context.Context, backoff time.Duration, attempt int) error {
	delay := backoff * time.Duration(attempt+1)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type modelHTTPError struct {
	status int
}

func (e *modelHTTPError) Error() string {
	return fmt.Sprintf("model provider returned status %d", e.status)
}

func isRetryableModelError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	var httpErr *modelHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.status == http.StatusTooManyRequests || httpErr.status >= http.StatusInternalServerError
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}

func modelResult(err error) string {
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "failure"
}
