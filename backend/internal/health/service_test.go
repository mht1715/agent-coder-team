package health

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func TestReady(t *testing.T) {
	t.Run("all dependencies are up", func(t *testing.T) {
		service := New(fakePinger{}, fakePinger{})

		result := service.Ready(context.Background())
		if result.Status != "ok" {
			t.Fatalf("expected status ok, got %s", result.Status)
		}
		if result.Checks["database"] != "up" || result.Checks["redis"] != "up" {
			t.Fatalf("unexpected checks: %#v", result.Checks)
		}
	})

	t.Run("reports degraded dependencies", func(t *testing.T) {
		service := New(fakePinger{err: errors.New("db down")}, fakePinger{})

		result := service.Ready(context.Background())
		if result.Status != "degraded" {
			t.Fatalf("expected status degraded, got %s", result.Status)
		}
		if result.Checks["database"] != "down" {
			t.Fatalf("expected database down, got %s", result.Checks["database"])
		}
		if result.Checks["redis"] != "up" {
			t.Fatalf("expected redis up, got %s", result.Checks["redis"])
		}
	})
}
