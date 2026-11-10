package data

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// isolateConfigDir points the settings machinery at a throwaway config
// directory for the lifetime of the test. Setting both HOME and
// XDG_CONFIG_HOME covers the non-Linux (~/.golazo) and Linux
// ($XDG_CONFIG_HOME/golazo) resolution rules of the dirs toolkit.
func isolateConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestNormalizeLeagueIDs(t *testing.T) {
	t.Run("empty selection falls back to sorted defaults", func(t *testing.T) {
		got := NormalizeLeagueIDs(nil)
		want := append([]int(nil), DefaultLeagueIDs...)
		slices.Sort(want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("NormalizeLeagueIDs(nil) = %v, want %v", got, want)
		}
		if !slices.IsSorted(got) {
			t.Errorf("default snapshot must be sorted, got %v", got)
		}
	})

	t.Run("deduplicates and sorts", func(t *testing.T) {
		got := NormalizeLeagueIDs([]int{87, 47, 47, 42, 87, 77})
		want := []int{42, 47, 77, 87}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("NormalizeLeagueIDs() = %v, want %v", got, want)
		}
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		got := NormalizeLeagueIDs(nil)
		got[0] = -1
		if slices.Contains(DefaultLeagueIDs, -1) {
			t.Fatal("mutating the snapshot must not mutate DefaultLeagueIDs")
		}
	})
}

// TestLoadActiveLeagueIDs_PreferenceSwitch verifies that each snapshot load
// reflects the settings currently on disk: after the preference file is
// rewritten, the next load must return the new league set rather than a
// stale one, and an empty selection resolves to the default leagues.
func TestLoadActiveLeagueIDs_PreferenceSwitch(t *testing.T) {
	isolateConfigDir(t)

	// First preference: an unsorted, duplicated selection that must come
	// back normalized.
	if err := SaveSettings(&Settings{SelectedLeagues: []int{87, 47, 87}}); err != nil {
		t.Fatalf("save first preference: %v", err)
	}
	got, err := LoadActiveLeagueIDs()
	if err != nil {
		t.Fatalf("LoadActiveLeagueIDs() error: %v", err)
	}
	if want := []int{47, 87}; !reflect.DeepEqual(got, want) {
		t.Errorf("first snapshot = %v, want %v", got, want)
	}

	// Same client path, switched preference: the new file must win.
	if err := SaveSettings(&Settings{SelectedLeagues: []int{42, 77}}); err != nil {
		t.Fatalf("save switched preference: %v", err)
	}
	got, err = LoadActiveLeagueIDs()
	if err != nil {
		t.Fatalf("LoadActiveLeagueIDs() after switch error: %v", err)
	}
	if want := []int{42, 77}; !reflect.DeepEqual(got, want) {
		t.Errorf("switched snapshot = %v, want %v", got, want)
	}

	// Empty selection: normalize to defaults instead of an empty snapshot.
	if err := SaveSettings(&Settings{SelectedLeagues: nil}); err != nil {
		t.Fatalf("save empty preference: %v", err)
	}
	got, err = LoadActiveLeagueIDs()
	if err != nil {
		t.Fatalf("LoadActiveLeagueIDs() empty error: %v", err)
	}
	wantDefault := append([]int(nil), DefaultLeagueIDs...)
	slices.Sort(wantDefault)
	if !reflect.DeepEqual(got, wantDefault) {
		t.Errorf("empty snapshot = %v, want defaults %v", got, wantDefault)
	}
}

// TestSaveSettings_AtomicRoundTrip verifies a successful atomic save lands
// the exact payload on disk and leaves no temporary files behind.
func TestSaveSettings_AtomicRoundTrip(t *testing.T) {
	dir := isolateConfigDir(t)

	want := &Settings{SelectedLeagues: []int{55, 47, 99}}
	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings() error: %v", err)
	}

	path, err := SettingsPath()
	if err != nil {
		t.Fatalf("SettingsPath() error: %v", err)
	}
	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}
	if !reflect.DeepEqual(loaded.SelectedLeagues, want.SelectedLeagues) {
		t.Errorf("loaded leagues = %v, want %v", loaded.SelectedLeagues, want.SelectedLeagues)
	}

	// A second save must replace the file in place, not accumulate temp files.
	if err := SaveSettings(&Settings{SelectedLeagues: []int{47}}); err != nil {
		t.Fatalf("second SaveSettings() error: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir() error: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".settings.yaml.tmp-") {
			t.Errorf("temporary file left behind after atomic save: %s", entry.Name())
		}
	}

	// Sanity: the config dir used here really is the isolated one.
	if !strings.HasPrefix(path, dir) {
		t.Errorf("settings path %q is outside isolated config dir %q", path, dir)
	}
}

// TestSaveSettings_WriteFailureFixture forces the final rename to fail by
// placing a directory at the settings path. The save must return an error,
// must not replace the directory with a half-written file, and must clean up
// its temporary file.
func TestSaveSettings_WriteFailureFixture(t *testing.T) {
	isolateConfigDir(t)

	path, err := SettingsPath()
	if err != nil {
		t.Fatalf("SettingsPath() error: %v", err)
	}
	// Directory at the target path makes os.Rename fail with EISDIR on both
	// macOS and Linux — a deterministic write-failure fixture.
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("create directory fixture: %v", err)
	}

	err = SaveSettings(&Settings{SelectedLeagues: []int{47, 87}})
	if err == nil {
		t.Fatal("SaveSettings() error = nil, want failure when target is a directory")
	}

	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("target path disappeared after failed save: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatal("failed save replaced the directory fixture with a file")
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir() error: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".settings.yaml.tmp-") {
			t.Errorf("temporary file leaked after failed save: %s", entry.Name())
		}
	}
}
