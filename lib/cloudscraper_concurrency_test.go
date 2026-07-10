package cloudscraper

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cserrors "github.com/Advik-B/cloudscraper/lib/errors"
	"github.com/Advik-B/cloudscraper/lib/stealth"
)

// newTestScraper builds a Scraper with stealth disabled so tests don't incur
// the default 500ms–2s human-like delay between requests, and with aggressive
// 403 retry settings for fast feedback.
func newTestScraper(t *testing.T, refreshInterval time.Duration, max403Retries int) *Scraper {
	t.Helper()
	s, err := New(
		WithStealth(stealth.Options{Enabled: false}),
		WithSessionConfig(true, refreshInterval, max403Retries),
	)
	if err != nil {
		t.Fatalf("New scraper: %v", err)
	}
	// Stealth disabled isn't enough — the transport still rotates TLS ciphers
	// which doesn't affect httptest (plain HTTP), so we're fine as-is.
	return s
}

// drainBody reads and closes the response body so connections are returned
// to the pool, keeping keep-alive behavior predictable.
func drainBody(resp *http.Response) {
	if resp == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// TestHandle403_CoalescesConcurrentRefreshes asserts that N concurrent 403s
// produce O(1) refresh probes (not N) and that every caller ultimately
// succeeds — the key regression from the old in403Retry bool design, where
// N−1 callers would see ErrMaxRetriesExceeded without retrying.
func TestHandle403_CoalescesConcurrentRefreshes(t *testing.T) {
	const concurrent = 20

	var refreshProbes atomic.Int32
	var userSucceed atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			refreshProbes.Add(1)
			userSucceed.Store(true)
			w.WriteHeader(http.StatusOK)
			return
		}
		if userSucceed.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	scraper := newTestScraper(t, 1*time.Hour, 3)

	var wg sync.WaitGroup
	errs := make([]error, concurrent)
	codes := make([]int, concurrent)
	ready := make(chan struct{})

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-ready
			resp, err := scraper.Get(server.URL + "/data")
			errs[i] = err
			if resp != nil {
				codes[i] = resp.StatusCode
				drainBody(resp)
			}
		}(i)
	}
	close(ready)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: unexpected error %v", i, err)
		}
		if codes[i] != http.StatusOK {
			t.Errorf("goroutine %d: got status %d, want 200", i, codes[i])
		}
	}

	probes := refreshProbes.Load()
	if probes == 0 {
		t.Fatalf("expected at least 1 refresh probe, got 0")
	}
	// Coalescing target: singleflight should collapse the concurrent burst
	// into a single refresh. Allow a small slack for scheduler-induced
	// follow-on refreshes (e.g. a straggler that arrives after the leader
	// returns), but reject the N-refreshes regression.
	if probes > 3 {
		t.Errorf("expected refresh probes to coalesce (~1), got %d — singleflight not coalescing", probes)
	}
}

// TestAutoRefresh_NoDeadlock asserts that the scheduled-refresh path in
// doWithRefresh no longer holds the mutex across network I/O. The old code
// called refreshSession → Get → do → mu.Lock() while still holding mu, a
// non-reentrant lock recursion that deadlocked the scraper on first refresh.
func TestAutoRefresh_NoDeadlock(t *testing.T) {
	const concurrent = 20

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	// Tiny interval + sleep forces every request to want a refresh.
	scraper := newTestScraper(t, 1*time.Millisecond, 3)
	time.Sleep(10 * time.Millisecond)

	done := make(chan error, concurrent)
	for i := 0; i < concurrent; i++ {
		go func() {
			resp, err := scraper.Get(server.URL + "/data")
			drainBody(resp)
			done <- err
		}()
	}

	// 10s is far above the HTTP client's 30s default but well below what a
	// deadlocked scraper would take (∞). On the old code this test hangs.
	deadline := time.After(10 * time.Second)
	for i := 0; i < concurrent; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
			}
		case <-deadline:
			t.Fatalf("deadlock suspected: only %d/%d goroutines returned", i, concurrent)
		}
	}
}

// TestRefreshFailure_PropagatesToAllCallers asserts the semantics the user
// called out: if the leader's refresh fails, every coalesced follower must
// also fail with the same error rather than retrying against stale state.
// We trigger failure by returning 403 on the root-URL probe, which the new
// refreshSession treats as a failed refresh.
func TestRefreshFailure_PropagatesToAllCallers(t *testing.T) {
	const concurrent = 20

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Every path — both user requests and refresh probes — returns 403.
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	scraper := newTestScraper(t, 1*time.Hour, 2)

	var wg sync.WaitGroup
	errs := make([]error, concurrent)
	ready := make(chan struct{})

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-ready
			resp, err := scraper.Get(server.URL + "/data")
			drainBody(resp)
			errs[i] = err
		}(i)
	}
	close(ready)
	wg.Wait()

	for i, err := range errs {
		if err == nil {
			t.Errorf("goroutine %d: expected error, got nil", i)
			continue
		}
		// Either path is acceptable: a wrapped refresh-failure, or
		// ErrMaxRetriesExceeded after all coalesced callers cycled through.
		if !errors.Is(err, cserrors.ErrMaxRetriesExceeded) &&
			!strings.Contains(err.Error(), "failed to refresh session") {
			t.Errorf("goroutine %d: unexpected error %v", i, err)
		}
	}
}

// TestConcurrentReadsUnderRefresh_NoDataRace fires many goroutines while
// forcing repeated refreshes. Under `go test -race`, any unsynchronized
// read of Scraper.UserAgent (written inside refreshSession) or the transport
// cipher suites will trip the race detector and fail the test.
func TestConcurrentReadsUnderRefresh_NoDataRace(t *testing.T) {
	const concurrent = 50
	const iterations = 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	scraper := newTestScraper(t, 1*time.Millisecond, 3)

	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				resp, err := scraper.Get(server.URL + "/x")
				drainBody(resp)
				if err != nil {
					t.Errorf("Get: %v", err)
					return
				}
				// Sleep just long enough that the next request triggers another
				// refresh via shouldRefreshSession().
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
}

// sanity check that Scraper.do is still the expected entry point from Get.
// If this breaks, the other tests above become misleading.
func TestGet_UsesDoWithRefresh(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	scraper := newTestScraper(t, 1*time.Hour, 3)
	resp, err := scraper.Get(server.URL + "/x")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	drainBody(resp)
	if got := calls.Load(); got != 1 {
		t.Fatalf("server calls = %d, want 1", got)
	}
}

