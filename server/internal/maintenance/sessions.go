package maintenance

import (
	"context"
	"log"
	"time"
)

const sessionCleanupTimeout = 30 * time.Second

type SessionStore interface {
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}

type SessionCleaner struct {
	store    SessionStore
	interval time.Duration
}

func NewSessionCleaner(store SessionStore, interval time.Duration) *SessionCleaner {
	return &SessionCleaner{
		store:    store,
		interval: interval,
	}
}

func (c *SessionCleaner) Run(ctx context.Context) {
	c.runPass(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runPass(ctx)
		}
	}
}

func (c *SessionCleaner) RunOnce(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, sessionCleanupTimeout)
	defer cancel()

	return c.store.DeleteExpiredSessions(ctx)
}

func (c *SessionCleaner) runPass(ctx context.Context) {
	removed, err := c.RunOnce(ctx)
	if err != nil {
		log.Printf("session cleanup: %v", err)
		return
	}
	if removed > 0 {
		log.Printf("session cleanup: removed %d expired sessions", removed)
	}
}
