package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/0xjuanma/golazo/internal/data"
	"github.com/0xjuanma/golazo/internal/ui"
)

// isolateAppSettingsHome redirects settings resolution to a throwaway HOME so
// the test never touches the developer's real ~/.golazo/settings.yaml.
func isolateAppSettingsHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// TestSettingsView_SaveFailureStaysAndPreservesCheckboxes drives the settings
// view with a write-failure fixture (a directory occupying settings.yaml).
// The failed Enter must keep the user inside the settings view with every
// checkbox intact and surface an error; once the fixture is repaired, the
// next Enter saves atomically and only then switches back to the main view.
func TestSettingsView_SaveFailureStaysAndPreservesCheckboxes(t *testing.T) {
	isolateAppSettingsHome(t)

	settingsPath, err := data.SettingsPath()
	if err != nil {
		t.Fatalf("SettingsPath() error: %v", err)
	}
	// Deterministic write-failure fixture: rename onto a directory fails.
	if err := os.MkdirAll(settingsPath, 0755); err != nil {
		t.Fatalf("create directory fixture: %v", err)
	}

	state := ui.NewSettingsState()
	state.Selected = map[int]bool{47: true, 87: true}
	state.HasChanges = true

	m := model{
		currentView:   viewSettings,
		settingsState: state,
		logger:        testLogger(),
	}

	// First Enter: persistence fails, so the view must not switch scope.
	next, cmd := m.handleSettingsViewKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("failed save returned a cmd, want nil")
	}
	failed := next.(model)
	if failed.currentView != viewSettings {
		t.Errorf("currentView after failed save = %v, want viewSettings", failed.currentView)
	}
	if failed.settingsState == nil {
		t.Fatal("settingsState was discarded on failed save; checkboxes are lost")
	}
	if !failed.settingsState.Selected[47] || !failed.settingsState.Selected[87] {
		t.Errorf("checkboxes not preserved: %+v", failed.settingsState.Selected)
	}
	if failed.settingsState.SaveError == "" {
		t.Error("SaveError is empty, want a visible persistence error")
	}
	// Nothing regular-file-like should have replaced the directory fixture.
	if info, err := os.Stat(settingsPath); err != nil || !info.IsDir() {
		t.Fatalf("settings path fixture altered after failed save: info=%v err=%v", info, err)
	}

	// Repair the fixture and retry: save now succeeds and the view exits.
	if err := os.Remove(settingsPath); err != nil {
		t.Fatalf("remove directory fixture: %v", err)
	}
	next, _ = failed.handleSettingsViewKeys(tea.KeyMsg{Type: tea.KeyEnter})
	saved := next.(model)
	if saved.currentView != viewMain {
		t.Errorf("currentView after successful save = %v, want viewMain", saved.currentView)
	}
	if saved.settingsState != nil {
		t.Error("settingsState should be cleared after a successful save")
	}

	loaded, err := data.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() after retry: %v", err)
	}
	if len(loaded.SelectedLeagues) != 2 {
		t.Fatalf("saved leagues = %v, want the two checked IDs", loaded.SelectedLeagues)
	}
	for _, want := range []int{47, 87} {
		found := false
		for _, got := range loaded.SelectedLeagues {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("saved selection missing league %d: %v", want, loaded.SelectedLeagues)
		}
	}

	// No temp files may linger in the config directory after either attempt.
	entries, err := os.ReadDir(filepath.Dir(settingsPath))
	if err != nil {
		t.Fatalf("ReadDir() error: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".settings.yaml.tmp-") {
			t.Errorf("temporary settings file leaked: %s", entry.Name())
		}
	}
}

// TestSettingsView_ToggleClearsSaveError verifies a new edit dismisses the
// stale failure banner so the user gets accurate feedback while retrying.
func TestSettingsView_ToggleClearsSaveError(t *testing.T) {
	isolateAppSettingsHome(t)

	state := ui.NewSettingsState()
	state.SaveError = "simulated persistence failure"
	state.Toggle() // any successful toggle clears the banner

	if state.SaveError != "" {
		t.Errorf("SaveError after toggle = %q, want empty", state.SaveError)
	}
	if !state.HasChanges {
		t.Error("HasChanges should be set after a toggle")
	}
}
