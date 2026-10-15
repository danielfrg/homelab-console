package apps

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

type testSource struct {
	apps  []App
	err   error
	calls int
}

func (s *testSource) List(context.Context) ([]App, error) {
	s.calls++
	return s.apps, s.err
}

func TestCatalogCacheRefreshAndStaleFallback(t *testing.T) {
	source := &testSource{apps: []App{{ID: "default/calendar-http", Name: "Calendar", Aliases: []string{}}}}
	catalog := NewCatalog(source)
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	catalog.now = func() time.Time { return now }
	first, err := catalog.Snapshot(context.Background(), false)
	if err != nil || len(first.Apps) != 1 || first.Stale || !first.UpdatedAt.Equal(now) {
		t.Fatalf("first snapshot = %+v, %v", first, err)
	}
	// Consumers cannot mutate the cached data or its nested aliases.
	first.Apps[0].Name = "Changed"
	first.Apps[0].Aliases = append(first.Apps[0].Aliases, "other.example.test")
	cached, err := catalog.Snapshot(context.Background(), false)
	if err != nil || source.calls != 1 || cached.Apps[0].Name != "Calendar" || len(cached.Apps[0].Aliases) != 0 {
		t.Fatalf("cached snapshot = %+v, calls %d, error %v", cached, source.calls, err)
	}
	source.err = errors.New("upstream unavailable")
	stale, err := catalog.Snapshot(context.Background(), true)
	if err != nil || source.calls != 2 || !stale.Stale || !stale.UpdatedAt.Equal(now) || stale.Apps[0].Name != "Calendar" {
		t.Fatalf("stale snapshot = %+v, calls %d, error %v", stale, source.calls, err)
	}
	now = now.Add(31 * time.Second)
	expired, err := catalog.Snapshot(context.Background(), false)
	if err != nil || source.calls != 3 || !expired.Stale {
		t.Fatalf("expired snapshot = %+v, calls %d, error %v", expired, source.calls, err)
	}
	source.err = nil
	source.apps = []App{{ID: "default/files-http", Name: "Files", Aliases: []string{}}}
	recovered, err := catalog.Snapshot(context.Background(), true)
	if err != nil || source.calls != 4 || recovered.Stale || recovered.Apps[0].Name != "Files" || !recovered.UpdatedAt.Equal(now) {
		t.Fatalf("recovered snapshot = %+v, calls %d, error %v", recovered, source.calls, err)
	}
}

func TestCatalogEmptySuccessAndFirstFailure(t *testing.T) {
	source := &testSource{}
	catalog := NewCatalog(source)
	empty, err := catalog.Snapshot(context.Background(), false)
	if err != nil || empty.Apps == nil || len(empty.Apps) != 0 || empty.UpdatedAt.IsZero() || empty.Stale {
		t.Fatalf("empty success = %+v, %v", empty, err)
	}
	source.err = errors.New("secret detail")
	stale, err := catalog.Snapshot(context.Background(), true)
	if err != nil || stale.Apps == nil || !stale.Stale {
		t.Fatalf("empty stale = %+v, %v", stale, err)
	}
	failed, err := NewCatalog(source).Snapshot(context.Background(), false)
	if err == nil || !reflect.DeepEqual(failed, Snapshot{}) {
		t.Fatalf("initial failure = %+v, %v", failed, err)
	}
}

func TestCatalogHonorsCancellation(t *testing.T) {
	source := &testSource{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewCatalog(source).Snapshot(ctx, false)
	if !errors.Is(err, context.Canceled) || source.calls != 0 {
		t.Fatalf("cancelled result = %v, calls %d", err, source.calls)
	}
}

func TestCatalogRemembersFailedRefreshWithinCacheTTL(t *testing.T) {
	source := &testSource{}
	catalog := NewCatalog(source)
	if _, err := catalog.Snapshot(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	source.err = errors.New("offline")
	if _, err := catalog.Snapshot(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	next, err := catalog.Snapshot(context.Background(), false)
	if err != nil || !next.Stale || source.calls != 2 {
		t.Fatalf("after failure snapshot = %+v, calls %d, error %v", next, source.calls, err)
	}
}

func TestCatalogConcurrentReadsShareSuccess(t *testing.T) {
	source := &testSource{}
	catalog := NewCatalog(source)
	var readers sync.WaitGroup
	for i := 0; i < 16; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			result, err := catalog.Snapshot(context.Background(), false)
			if err != nil || result.Apps == nil || result.Stale {
				t.Errorf("snapshot = %+v, %v", result, err)
			}
		}()
	}
	readers.Wait()
	if source.calls != 1 {
		t.Fatalf("calls = %d, want 1", source.calls)
	}
}

type blockingSource struct{ started, release chan struct{} }

func (s *blockingSource) List(ctx context.Context) ([]App, error) {
	close(s.started)
	select {
	case <-s.release:
		return []App{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestCatalogCancellationWhileWaitingForRefresh(t *testing.T) {
	source := &blockingSource{started: make(chan struct{}), release: make(chan struct{})}
	catalog := NewCatalog(source)
	firstFinished := make(chan struct{})
	go func() { defer close(firstFinished); _, _ = catalog.Snapshot(context.Background(), false) }()
	<-source.started
	defer func() { close(source.release); <-firstFinished }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := catalog.Snapshot(ctx, false)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait error = %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("waiting for another refresh ignored caller's deadline")
	}
}
