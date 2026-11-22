package health

import (
	"context"
	"sync"
	"time"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type CheckResult struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type Service struct {
	db    Pinger
	cache Pinger
}

func New(db, cache Pinger) *Service {
	return &Service{db: db, cache: cache}
}

func (s *Service) Ready(ctx context.Context) CheckResult {
	result := CheckResult{
		Status: "ok",
		Checks: map[string]string{
			"database": "up",
			"redis":    "up",
		},
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	check := func(name string, pinger Pinger) {
		defer wg.Done()

		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := pinger.Ping(checkCtx); err != nil {
			mu.Lock()
			result.Status = "degraded"
			result.Checks[name] = "down"
			mu.Unlock()
		}
	}

	wg.Add(2)
	go check("database", s.db)
	go check("redis", s.cache)
	wg.Wait()

	return result
}
