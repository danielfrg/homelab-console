package cluster

import (
	"context"
	"time"

	"github.com/danielfrg/homelab-console/internal/apps"
)

const snapshotTTL = 15 * time.Second

type LinkSource interface {
	Snapshot(context.Context, bool) (apps.Snapshot, error)
}

// Inventory serializes reads and keeps the last good snapshot, marking it stale
// after a failed refresh, like the launcher's route catalog.
type Inventory struct {
	reader Reader
	links  LinkSource
	gate   chan struct{}
	cached Snapshot
	fresh  bool
	now    func() time.Time
}

func NewInventory(reader Reader, links LinkSource) *Inventory {
	return &Inventory{reader: reader, links: links, gate: make(chan struct{}, 1), now: time.Now}
}

func (i *Inventory) Snapshot(ctx context.Context, refresh bool) (Snapshot, error) {
	select {
	case i.gate <- struct{}{}:
		defer func() { <-i.gate }()
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
	if !refresh && i.fresh && i.now().Sub(i.cached.UpdatedAt) < snapshotTTL {
		return i.cached, nil
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*requestTimeout)
	defer cancel()
	raw, err := i.reader.Read(requestCtx)
	if err != nil {
		if i.cached.UpdatedAt.IsZero() {
			return Snapshot{}, err
		}
		i.cached.Stale = true
		return i.cached, nil
	}
	if i.links != nil {
		if links, err := i.links.Snapshot(requestCtx, refresh); err == nil {
			raw.Links = links.Apps
		}
	}
	i.cached = Build(raw, i.now())
	i.fresh = true
	return i.cached, nil
}

// Invalidate makes the next read go to the cluster, used after a restart.
func (i *Inventory) Invalidate() {
	i.gate <- struct{}{}
	i.fresh = false
	<-i.gate
}
