package apps

import (
	"context"
	"time"
)

const (
	cacheTTL       = 30 * time.Second
	requestTimeout = 5 * time.Second
)

// App contains only the presentation fields needed by the launchpad.
type App struct {
	ID          string   `json:"id" doc:"Namespace and HTTPRoute name"`
	Name        string   `json:"name"`
	URL         string   `json:"url" format:"uri"`
	Aliases     []string `json:"aliases" nullable:"false" doc:"Additional internal hostnames"`
	Description string   `json:"description"`
	Group       string   `json:"group"`
	Icon        string   `json:"icon" doc:"Optional icon identifier"`
}

type Snapshot struct {
	Apps      []App     `json:"apps" nullable:"false"`
	UpdatedAt time.Time `json:"updatedAt" doc:"Time of the last successful discovery"`
	Stale     bool      `json:"stale" doc:"Last successful discovery retained after a refresh failure"`
}

type Source interface {
	List(context.Context) ([]App, error)
}

// Catalog serializes refreshes and retains the last successful small snapshot.
// It starts no background goroutines or watches.
type Catalog struct {
	source Source
	gate   chan struct{}
	cached Snapshot
	now    func() time.Time
}

func NewCatalog(source Source) *Catalog {
	return &Catalog{source: source, now: time.Now, gate: make(chan struct{}, 1)}
}

func (c *Catalog) Snapshot(ctx context.Context, refresh bool) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	select {
	case c.gate <- struct{}{}:
		defer func() { <-c.gate }()
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if !refresh && !c.cached.UpdatedAt.IsZero() && c.now().Sub(c.cached.UpdatedAt) < cacheTTL {
		return cloneSnapshot(c.cached), nil
	}
	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	discovered, err := c.source.List(requestCtx)
	if err != nil {
		if c.cached.UpdatedAt.IsZero() {
			return Snapshot{}, err
		}
		c.cached.Stale = true
		return cloneSnapshot(c.cached), nil
	}
	c.cached = cloneSnapshot(Snapshot{Apps: discovered, UpdatedAt: c.now().UTC()})
	return cloneSnapshot(c.cached), nil
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	result := snapshot
	result.Apps = make([]App, len(snapshot.Apps))
	for i, app := range snapshot.Apps {
		result.Apps[i] = app
		result.Apps[i].Aliases = append([]string{}, app.Aliases...)
	}
	return result
}
