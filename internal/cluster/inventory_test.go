package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danielfrg/homelab-console/internal/apps"
)

type stubReader struct {
	reads int
	err   error
}

func (r *stubReader) Read(context.Context) (Raw, error) {
	r.reads++
	if r.err != nil {
		return Raw{}, r.err
	}
	raw := fixture()
	raw.Links = nil
	return raw, nil
}

type stubLinks struct {
	snapshot apps.Snapshot
	err      error
}

func (l stubLinks) Snapshot(context.Context, bool) (apps.Snapshot, error) { return l.snapshot, l.err }

func TestInventoryTTLAndStale(t *testing.T) {
	reader := &stubReader{}
	now := testNow
	inv := NewInventory(reader, stubLinks{snapshot: apps.Snapshot{Apps: fixture().Links}})
	inv.now = func() time.Time { return now }
	ctx := context.Background()

	s, err := inv.Snapshot(ctx, false)
	if err != nil || reader.reads != 1 || len(s.Apps) == 0 || s.Stale {
		t.Fatal(s, err)
	}
	if ha, _, _ := s.FindApp("home-assistant"); len(ha.Links) != 1 {
		t.Fatalf("links not joined %+v", ha.Links)
	}
	now = now.Add(10 * time.Second)
	if _, err = inv.Snapshot(ctx, false); err != nil || reader.reads != 1 {
		t.Fatal("expected cached read")
	}
	now = now.Add(10 * time.Second)
	reader.err = errors.New("down")
	s, err = inv.Snapshot(ctx, false)
	if err != nil || !s.Stale || len(s.Apps) == 0 || reader.reads != 2 {
		t.Fatal(s.Stale, err)
	}
	reader.err = nil
	inv.Invalidate()
	if s, _ = inv.Snapshot(ctx, false); reader.reads != 3 || s.Stale {
		t.Fatal("invalidate should force a fresh read")
	}
	if _, _ = inv.Snapshot(ctx, true); reader.reads != 4 {
		t.Fatal("refresh should force a read")
	}

	fresh := NewInventory(&stubReader{err: errors.New("down")}, stubLinks{err: errors.New("x")})
	if _, err = fresh.Snapshot(ctx, false); err == nil {
		t.Fatal("first failure should error")
	}
	linkless := NewInventory(&stubReader{}, stubLinks{err: errors.New("routes down")})
	if s, err = linkless.Snapshot(ctx, false); err != nil || len(s.Apps) == 0 {
		t.Fatal("link failure must not fail the snapshot", err)
	}
}
