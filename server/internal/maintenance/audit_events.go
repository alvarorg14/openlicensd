package maintenance

import (
	"context"
	"log/slog"
	"time"
)

const auditEventPruneTimeout = 30 * time.Second

type AuditEventStore interface {
	DeleteAuditEventsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type AuditEventPruner struct {
	store         AuditEventStore
	retention     time.Duration
	interval      time.Duration
	logger        *slog.Logger
}

func NewAuditEventPruner(store AuditEventStore, retentionDays int, interval time.Duration, logger *slog.Logger) *AuditEventPruner {
	return &AuditEventPruner{
		store:     store,
		retention: time.Duration(retentionDays) * 24 * time.Hour,
		interval:  interval,
		logger:    logger,
	}
}

func (p *AuditEventPruner) Run(ctx context.Context) {
	p.runPass(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.runPass(ctx)
		}
	}
}

func (p *AuditEventPruner) RunOnce(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, auditEventPruneTimeout)
	defer cancel()

	cutoff := time.Now().UTC().Add(-p.retention)
	return p.store.DeleteAuditEventsBefore(ctx, cutoff)
}

func (p *AuditEventPruner) runPass(ctx context.Context) {
	removed, err := p.RunOnce(ctx)
	if err != nil {
		p.logger.Error("audit event prune failed", slog.Any("err", err))
		return
	}
	if removed > 0 {
		p.logger.Info("audit event prune completed", slog.Int64("removed", removed))
	}
}
