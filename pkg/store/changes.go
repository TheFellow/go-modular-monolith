package store

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

const DefaultChangePollInterval = 250 * time.Millisecond

// ChangeMonitor turns SQLite's connection-local data_version into a
// process-local, coalesced invalidation signal. Signals deliberately carry no
// data: consumers must re-query through the application layer.
type ChangeMonitor struct {
	db       *sql.DB
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
	signals  chan struct{}
	epoch    atomic.Uint64
	close    sync.Once
}

// MonitorChanges pins one read connection and begins watching commits made by
// every other connection, including connections in other processes.
func (s *Store) MonitorChanges(ctx context.Context, interval time.Duration) (*ChangeMonitor, error) {
	if interval <= 0 {
		interval = DefaultChangePollInterval
	}
	watchCtx, cancel := context.WithCancel(ctx)
	conn, version, err := openChangeConnection(watchCtx, s.db)
	if err != nil {
		cancel()
		return nil, err
	}
	m := &ChangeMonitor{
		db: s.db, interval: interval, cancel: cancel,
		done: make(chan struct{}), signals: make(chan struct{}, 1),
	}
	go m.run(watchCtx, conn, version)
	return m, nil
}

// Signals is edge notification for an epoch change. The channel is
// intentionally coalesced; Epoch is the level that consumers compare around a
// refresh to avoid lost wakeups.
func (m *ChangeMonitor) Signals() <-chan struct{} { return m.signals }

func (m *ChangeMonitor) Epoch() uint64 { return m.epoch.Load() }

func (m *ChangeMonitor) Done() <-chan struct{} { return m.done }

func (m *ChangeMonitor) Close() {
	if m == nil {
		return
	}
	m.close.Do(func() {
		m.cancel()
		<-m.done
	})
}

func (m *ChangeMonitor) run(ctx context.Context, conn *sql.Conn, version uint64) {
	defer close(m.done)
	defer func() { _ = conn.Close() }()
	timer := time.NewTimer(m.interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		current, err := dataVersion(ctx, conn)
		if err != nil {
			_ = conn.Close()
			conn, version, err = m.reconnect(ctx)
			if err != nil {
				return
			}
			// Changes may have committed while the old connection was unusable.
			m.publish()
		} else if current != version {
			version = current
			m.publish()
		}
		timer.Reset(m.interval)
	}
}

func (m *ChangeMonitor) reconnect(ctx context.Context) (*sql.Conn, uint64, error) {
	delay := 25 * time.Millisecond
	for {
		conn, version, err := openChangeConnection(ctx, m.db)
		if err == nil {
			return conn, version, nil
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, 0, ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, time.Second)
	}
}

func (m *ChangeMonitor) publish() {
	m.epoch.Add(1)
	select {
	case m.signals <- struct{}{}:
	default:
	}
}

func openChangeConnection(ctx context.Context, db *sql.DB) (*sql.Conn, uint64, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, 0, errors.Internalf("acquire sqlite change connection: %w", err)
	}
	version, err := dataVersion(ctx, conn)
	if err != nil {
		_ = conn.Close()
		return nil, 0, errors.Internalf("read sqlite data version: %w", err)
	}
	return conn, version, nil
}

func dataVersion(ctx context.Context, conn *sql.Conn) (uint64, error) {
	var version uint64
	err := conn.QueryRowContext(ctx, "PRAGMA data_version").Scan(&version)
	return version, err
}
