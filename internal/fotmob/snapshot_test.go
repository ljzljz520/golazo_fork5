package fotmob

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xjuanma/golazo/internal/api"
	"github.com/0xjuanma/golazo/internal/data"
	"github.com/0xjuanma/golazo/internal/ratelimit"
)

// --- Fixture builders -------------------------------------------------------

type fixtureStatus struct {
	UTCTime   string `json:"utcTime"`
	Started   *bool  `json:"started,omitempty"`
	Finished  *bool  `json:"finished,omitempty"`
	Cancelled *bool  `json:"cancelled,omitempty"`
}

type fixtureTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type fixtureMatch struct {
	ID      string        `json:"id"`
	Home    fixtureTeam   `json:"home"`
	Away    fixtureTeam   `json:"away"`
	Status  fixtureStatus `json:"status"`
	PageURL string        `json:"pageUrl"`
}

func fixtureBoolPtr(v bool) *bool { return &v }

// newFixtureMatch builds a minimal league-page match on the given UTC date.
func newFixtureMatch(id int, utcDate string, finished, started bool) fixtureMatch {
	return fixtureMatch{
		ID:   fmt.Sprintf("%d", id),
		Home: fixtureTeam{ID: fmt.Sprintf("h%d", id), Name: fmt.Sprintf("Home %d", id)},
		Away: fixtureTeam{ID: fmt.Sprintf("a%d", id), Name: fmt.Sprintf("Away %d", id)},
		Status: fixtureStatus{
			UTCTime:  utcDate + "T15:00:00Z",
			Started:  fixtureBoolPtr(started),
			Finished: fixtureBoolPtr(finished),
		},
		PageURL: fmt.Sprintf("/matches/home-%d-vs-away-%d/fix%d", id, id, id),
	}
}

// leaguePageHTML renders a Next.js page shell whose pageProps carry the
// supplied league details and fixture list.
func leaguePageHTML(t *testing.T, leagueID int, matches []fixtureMatch) string {
	t.Helper()
	props := struct {
		Details struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"details"`
		Fixtures struct {
			AllMatches []fixtureMatch `json:"allMatches"`
		} `json:"fixtures"`
	}{}
	props.Details.ID = leagueID
	props.Details.Name = fmt.Sprintf("League %d", leagueID)
	props.Details.Country = "FX"
	props.Fixtures.AllMatches = matches

	pageProps, err := json.Marshal(props)
	if err != nil {
		t.Fatalf("marshal page props: %v", err)
	}
	wrapper, err := json.Marshal(struct {
		Props json.RawMessage `json:"props"`
	}{Props: mustRawMessage(t, struct {
		PageProps json.RawMessage `json:"pageProps"`
	}{PageProps: pageProps})})
	if err != nil {
		t.Fatalf("marshal wrapper: %v", err)
	}
	return `<html><script id="__NEXT_DATA__" type="application/json">` + string(wrapper) + `</script></html>`
}

func mustRawMessage(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal raw message: %v", err)
	}
	return raw
}

// recordingTransport serves canned league pages and records every league ID
// requested over the wire.
type recordingTransport struct {
	mu       sync.Mutex
	pages    map[int]string
	hits     map[int]int
	requests []int
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	leagueID := extractIDFromPath(req.URL.Path)

	rt.mu.Lock()
	rt.requests = append(rt.requests, leagueID)
	rt.hits[leagueID]++
	body, ok := rt.pages[leagueID]
	rt.mu.Unlock()

	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Request:    req,
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
		Header:     make(http.Header),
	}, nil
}

func (rt *recordingTransport) requestedLeagues() []int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	seen := make(map[int]struct{})
	var ids []int
	for _, id := range rt.requests {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (rt *recordingTransport) hitCount(leagueID int) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.hits[leagueID]
}

func newSnapshotTestClient(t *testing.T, pages map[int]string) (*Client, *recordingTransport) {
	t.Helper()
	transport := &recordingTransport{
		pages: pages,
		hits:  make(map[int]int),
	}
	client := &Client{
		httpClient:    &http.Client{Transport: transport, Timeout: 5 * time.Second},
		baseURL:       baseURL,
		rateLimiter:   ratelimit.New(0),
		cache:         NewResponseCache(DefaultCacheConfig()),
		pageURLs:      make(map[int]string, 10),
		maxConcurrent: make(chan struct{}, 10),
	}
	return client, transport
}

func isolateSettingsHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
}

func matchIDs(matches []api.Match) []int {
	ids := make([]int, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m.ID)
	}
	sort.Ints(ids)
	return ids
}

// --- Cache key isolation ----------------------------------------------------

func TestBuildMatchesCacheKey_Isolation(t *testing.T) {
	full := []string{"fixtures", "results"}

	keyA := BuildMatchesCacheKey([]int{47, 87}, "2026-03-10", full)
	keyB := BuildMatchesCacheKey([]int{87, 47}, "2026-03-10", full) // different order, same set
	if keyA != keyB {
		t.Errorf("league order changed the key: %+v != %+v", keyA, keyB)
	}

	cases := []struct {
		name    string
		leagues []int
		date    string
		tabs    []string
	}{
		{"different league set", []int{42, 77}, "2026-03-10", full},
		{"different date", []int{47, 87}, "2026-03-11", full},
		{"results only", []int{47, 87}, "2026-03-10", []string{"results"}},
		{"fixtures only", []int{47, 87}, "2026-03-10", []string{"fixtures"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildMatchesCacheKey(tc.leagues, tc.date, tc.tabs)
			if got == keyA {
				t.Errorf("key for %s collides with the full baseline key: %+v", tc.name, got)
			}
		})
	}

	// Tab order must not matter either.
	reversedTabs := BuildMatchesCacheKey([]int{47, 87}, "2026-03-10", []string{"results", "fixtures"})
	if reversedTabs != keyA {
		t.Errorf("tab order changed the key: %+v != %+v", reversedTabs, keyA)
	}
}

// --- Batch slicing ----------------------------------------------------------

func TestBatchLeagueIDs(t *testing.T) {
	snapshot := []int{42, 47, 77, 87, 55}

	t.Run("walks every batch from the fixed snapshot", func(t *testing.T) {
		var collected []int
		index := 0
		for {
			ids, isLast := BatchLeagueIDs(snapshot, index, 2)
			collected = append(collected, ids...)
			wantLast := index == 2 // 5 leagues, size 2 -> batches 0,1,2
			if isLast != wantLast {
				t.Errorf("batch %d isLast = %v, want %v", index, isLast, wantLast)
			}
			if isLast {
				break
			}
			index++
			if index > 10 {
				t.Fatal("BatchLeagueIDs never reported isLast")
			}
		}
		if !reflect.DeepEqual(collected, snapshot) {
			t.Errorf("collected batches = %v, want snapshot %v", collected, snapshot)
		}
	})

	t.Run("single batch covers everything", func(t *testing.T) {
		ids, isLast := BatchLeagueIDs(snapshot, 0, len(snapshot))
		if !isLast {
			t.Error("isLast = false, want true for single batch")
		}
		if !reflect.DeepEqual(ids, snapshot) {
			t.Errorf("ids = %v, want %v", ids, snapshot)
		}
	})

	t.Run("out of range and bad inputs terminate", func(t *testing.T) {
		if ids, isLast := BatchLeagueIDs(snapshot, 99, 2); len(ids) != 0 || !isLast {
			t.Errorf("out-of-range batch = %v, isLast %v, want empty/last", ids, isLast)
		}
		if ids, isLast := BatchLeagueIDs(snapshot, 0, 0); len(ids) != 0 || !isLast {
			t.Errorf("zero batch size = %v, isLast %v, want empty/last", ids, isLast)
		}
		if ids, isLast := BatchLeagueIDs(nil, 0, 4); len(ids) != 0 || !isLast {
			t.Errorf("nil snapshot = %v, isLast %v, want empty/last", ids, isLast)
		}
	})
}

// --- Same client, switched preference --------------------------------------

// TestMatchesByDate_SameClientPreferenceSwitch drives one client through two
// preference rounds on the same date. The second round must not be served the
// first round's cached schedule, and switching back must return the original
// league's results.
func TestMatchesByDate_SameClientPreferenceSwitch(t *testing.T) {
	isolateSettingsHome(t)

	const dateKey = "2026-03-10"
	queryDate := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	pages := map[int]string{
		47: leaguePageHTML(t, 47, []fixtureMatch{newFixtureMatch(3001, dateKey, true, true)}),
		87: leaguePageHTML(t, 87, []fixtureMatch{newFixtureMatch(3002, dateKey, true, true)}),
	}
	client, transport := newSnapshotTestClient(t, pages)
	ctx := context.Background()
	fullTabs := []string{"fixtures", "results"}

	// Round 1: preference is league 47.
	if err := data.SaveSettings(&data.Settings{SelectedLeagues: []int{47}}); err != nil {
		t.Fatalf("save preference 47: %v", err)
	}
	round1, err := client.MatchesByDateWithTabs(ctx, queryDate, fullTabs)
	if err != nil {
		t.Fatalf("round 1 query: %v", err)
	}
	if got := matchIDs(round1); !reflect.DeepEqual(got, []int{3001}) {
		t.Errorf("round 1 matches = %v, want only league 47's match 3001", got)
	}

	// Round 2: same client, same date, preference switched to league 87. The
	// date-only cache of the old implementation would have returned 3001 here.
	if err := data.SaveSettings(&data.Settings{SelectedLeagues: []int{87}}); err != nil {
		t.Fatalf("save preference 87: %v", err)
	}
	round2, err := client.MatchesByDateWithTabs(ctx, queryDate, fullTabs)
	if err != nil {
		t.Fatalf("round 2 query: %v", err)
	}
	if got := matchIDs(round2); !reflect.DeepEqual(got, []int{3002}) {
		t.Errorf("round 2 matches = %v, want only league 87's match 3002 (stale cache?)", got)
	}
	if got := transport.requestedLeagues(); !reflect.DeepEqual(got, []int{47, 87}) {
		t.Errorf("requested leagues = %v, want [47 87]", got)
	}

	// Round 3: switch back to 47. The league-set-scoped cache must serve the
	// original 3001 result without contaminating it with round 2's data.
	if err := data.SaveSettings(&data.Settings{SelectedLeagues: []int{47}}); err != nil {
		t.Fatalf("re-save preference 47: %v", err)
	}
	round3, err := client.MatchesByDateWithTabs(ctx, queryDate, fullTabs)
	if err != nil {
		t.Fatalf("round 3 query: %v", err)
	}
	if got := matchIDs(round3); !reflect.DeepEqual(got, []int{3001}) {
		t.Errorf("round 3 matches = %v, want cached league 47 match 3001", got)
	}
}

// --- External settings change mid-query ------------------------------------

// TestSnapshot_ExternalSettingsChangeMidQuery fixes a snapshot, then rewrites
// the settings file before the round runs. Every batch and every network
// request must still derive from the original snapshot; a fresh round picks
// up the new file.
func TestSnapshot_ExternalSettingsChangeMidQuery(t *testing.T) {
	isolateSettingsHome(t)

	const dateKey = "2026-03-10"
	queryDate := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	pages := map[int]string{
		42: leaguePageHTML(t, 42, []fixtureMatch{newFixtureMatch(4001, dateKey, true, true)}),
		47: leaguePageHTML(t, 47, []fixtureMatch{newFixtureMatch(4002, dateKey, true, true)}),
		77: leaguePageHTML(t, 77, []fixtureMatch{newFixtureMatch(4003, dateKey, true, true)}),
		87: leaguePageHTML(t, 87, []fixtureMatch{newFixtureMatch(4004, dateKey, true, true)}),
	}

	// Start a round with preference A (stored unordered to prove normalization).
	if err := data.SaveSettings(&data.Settings{SelectedLeagues: []int{87, 47}}); err != nil {
		t.Fatalf("save preference A: %v", err)
	}
	snapshot, err := CaptureActiveLeagues()
	if err != nil {
		t.Fatalf("CaptureActiveLeagues() error: %v", err)
	}
	if want := []int{47, 87}; !reflect.DeepEqual(snapshot, want) {
		t.Fatalf("snapshot = %v, want normalized %v", snapshot, want)
	}

	// External actor rewrites the config to preference B before any batch runs.
	if err := data.SaveSettings(&data.Settings{SelectedLeagues: []int{42, 77}}); err != nil {
		t.Fatalf("external preference B rewrite: %v", err)
	}
	if got := data.ActiveLeagueIDs(); !reflect.DeepEqual(got, []int{42, 77}) {
		t.Fatalf("settings on disk = %v, want [42 77]", got)
	}

	// Batch math stays on snapshot A.
	var collected []int
	for i := 0; ; i++ {
		ids, isLast := BatchLeagueIDs(snapshot, i, 1)
		collected = append(collected, ids...)
		if isLast {
			break
		}
		if i > 10 {
			t.Fatal("batch walk did not terminate")
		}
	}
	if !reflect.DeepEqual(collected, []int{47, 87}) {
		t.Errorf("batches covered %v, want snapshot A [47 87]", collected)
	}

	// Aggregated date query on the fixed snapshot touches only A's leagues.
	client, transport := newSnapshotTestClient(t, pages)
	matches, err := client.MatchesByDateWithTabsForLeagues(
		context.Background(), queryDate, []string{"fixtures", "results"}, snapshot,
	)
	if err != nil {
		t.Fatalf("snapshot date query: %v", err)
	}
	if got := matchIDs(matches); !reflect.DeepEqual(got, []int{4002, 4004}) {
		t.Errorf("snapshot query matches = %v, want [4002 4004] from leagues 47/87", got)
	}
	if got := transport.requestedLeagues(); !reflect.DeepEqual(got, []int{47, 87}) {
		t.Errorf("snapshot query requested leagues %v, want [47 87]", got)
	}

	// Live aggregation on the same snapshot also stays on A.
	liveClient, liveTransport := newSnapshotTestClient(t, pages)
	if _, _, err := liveClient.LiveAndUpcomingForLeagues(context.Background(), snapshot); err != nil {
		t.Fatalf("LiveAndUpcomingForLeagues() error: %v", err)
	}
	if got := liveTransport.requestedLeagues(); !reflect.DeepEqual(got, []int{47, 87}) {
		t.Errorf("live snapshot requested leagues %v, want [47 87]", got)
	}

	// A brand-new round captures preference B — the snapshot boundary, not a
	// stale in-memory set, decides when the new scope takes effect.
	freshClient, freshTransport := newSnapshotTestClient(t, pages)
	if _, err := freshClient.MatchesByDateWithTabs(
		context.Background(), queryDate, []string{"fixtures", "results"},
	); err != nil {
		t.Fatalf("fresh round query: %v", err)
	}
	if got := freshTransport.requestedLeagues(); !reflect.DeepEqual(got, []int{42, 77}) {
		t.Errorf("fresh round requested leagues %v, want [42 77]", got)
	}
}

// --- Different status queries ----------------------------------------------

// TestMatchesByDate_StatusQueriesAreIsolated proves results-only and
// fixtures+results queries on the same date/leagues keep separate cache
// entries: a results-only response must never be served to a full query, and
// repeating the results-only query returns its own cached entry.
func TestMatchesByDate_StatusQueriesAreIsolated(t *testing.T) {
	const dateKey = "2026-03-10"
	queryDate := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	pages := map[int]string{
		47: leaguePageHTML(t, 47, []fixtureMatch{
			newFixtureMatch(5001, dateKey, true, true),   // finished -> "results"
			newFixtureMatch(5002, dateKey, false, false), // not started -> "fixtures"
		}),
	}
	client, _ := newSnapshotTestClient(t, pages)
	ctx := context.Background()
	snapshot := []int{47}

	// Results-only round.
	results, err := client.MatchesByDateWithTabsForLeagues(ctx, queryDate, []string{"results"}, snapshot)
	if err != nil {
		t.Fatalf("results query: %v", err)
	}
	if got := matchIDs(results); !reflect.DeepEqual(got, []int{5001}) {
		t.Errorf("results-only matches = %v, want [5001]", got)
	}

	// Full round must recompute (composite key differs) and include both.
	full, err := client.MatchesByDateWithTabsForLeagues(ctx, queryDate, []string{"fixtures", "results"}, snapshot)
	if err != nil {
		t.Fatalf("full query: %v", err)
	}
	if got := matchIDs(full); !reflect.DeepEqual(got, []int{5001, 5002}) {
		t.Errorf("full matches = %v, want [5001 5002] (results-only cache leaked?)", got)
	}

	// Repeating results-only returns its own isolated cached entry.
	resultsAgain, err := client.MatchesByDateWithTabsForLeagues(ctx, queryDate, []string{"results"}, snapshot)
	if err != nil {
		t.Fatalf("repeat results query: %v", err)
	}
	if got := matchIDs(resultsAgain); !reflect.DeepEqual(got, []int{5001}) {
		t.Errorf("repeat results matches = %v, want isolated [5001]", got)
	}

	// Both entries must exist independently in the cache.
	resultsKey := BuildMatchesCacheKey(snapshot, dateKey, []string{"results"})
	fullKey := BuildMatchesCacheKey(snapshot, dateKey, []string{"fixtures", "results"})
	if cached := client.cache.Matches(resultsKey); len(cached) != 1 {
		t.Errorf("cached results entry len = %d, want 1", len(cached))
	}
	if cached := client.cache.Matches(fullKey); len(cached) != 2 {
		t.Errorf("cached full entry len = %d, want 2", len(cached))
	}
}
