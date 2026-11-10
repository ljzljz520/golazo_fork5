package data

import (
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

const settingsFileName = "settings.yaml"

// LeagueInfo contains league metadata for display purposes.
type LeagueInfo struct {
	ID      int
	Name    string
	Country string
}

// Region constants for organizing leagues
const (
	RegionEurope  = "Europe"
	RegionAmerica = "Americas"
	RegionGlobal  = "Global"
)

// AllSupportedLeagues contains all leagues that Golazo supports organized by region.
// This is the source of truth for available leagues.
var AllSupportedLeagues = map[string][]LeagueInfo{
	RegionEurope: {
		// Top 5 European Leagues
		{ID: 47, Name: "Premier League", Country: "England"},
		{ID: 87, Name: "La Liga", Country: "Spain"},
		{ID: 140, Name: "La Liga 2", Country: "Spain"},
		{ID: 54, Name: "Bundesliga", Country: "Germany"},
		{ID: 146, Name: "2. Bundesliga", Country: "Germany"},
		{ID: 208, Name: "3. Liga", Country: "Germany"},
		{ID: 512, Name: "Regionalliga", Country: "Germany"},
		{ID: 55, Name: "Serie A", Country: "Italy"},
		{ID: 86, Name: "Serie B", Country: "Italy"},
		{ID: 53, Name: "Ligue 1", Country: "France"},
		{ID: 110, Name: "Ligue 2", Country: "France"},
		// Top 5 Women's Leagues
		{ID: 9227, Name: "Women's Super League", Country: "England"},
		{ID: 9907, Name: "Liga F", Country: "Spain"},
		{ID: 9676, Name: "Frauen-Bundesliga", Country: "Germany"},
		{ID: 10178, Name: "Serie A Femminile", Country: "Italy"},
		{ID: 9667, Name: "Première Ligue Féminine", Country: "France"},
		// Other European Leagues
		{ID: 67, Name: "Allsvenskan", Country: "Sweden"},
		{ID: 38, Name: "Austrian Bundesliga", Country: "Austria"},
		{ID: 40, Name: "Belgian First Division", Country: "Belgium"},
		{ID: 48, Name: "EFL Championship", Country: "England"},
		{ID: 108, Name: "EFL League One", Country: "England"},
		{ID: 109, Name: "EFL League Two", Country: "England"},
		{ID: 196, Name: "Ekstraklasa", Country: "Poland"},
		{ID: 57, Name: "Eredivisie", Country: "Netherlands"},
		{ID: 218, Name: "League of Ireland First Division", Country: "Ireland"},
		{ID: 126, Name: "League of Ireland Premier Division", Country: "Ireland"},
		{ID: 61, Name: "Primeira Liga", Country: "Portugal"},
		{ID: 10215, Name: "Primeira Liga Qualification", Country: "Portugal"},
		{ID: 185, Name: "Liga Portugal 2", Country: "Portugal"},
		{ID: 9668, Name: "Liga Portugal 2 Qualification", Country: "Portugal"},
		{ID: 64, Name: "Scottish Premiership", Country: "Scotland"},
		{ID: 135, Name: "Super League 1", Country: "Greece"},
		{ID: 46, Name: "Superligaen", Country: "Denmark"},
		{ID: 85, Name: "1. Division", Country: "Denmark"},
		{ID: 59, Name: "Eliteserien", Country: "Norway"},
		{ID: 203, Name: "1. Divisjon", Country: "Norway"},
		{ID: 71, Name: "Süper Lig", Country: "Turkey"},
		{ID: 69, Name: "Swiss Super League", Country: "Switzerland"},
		{ID: 63, Name: "Russian Premier League", Country: "Russia"},
		{ID: 441, Name: "Ukrainian Premier League", Country: "Ukraine"},
		// European Competitions
		{ID: 42, Name: "UEFA Champions League", Country: "Europe"},
		{ID: 10216, Name: "UEFA Conference League", Country: "Europe"},
		{ID: 73, Name: "UEFA Europa League", Country: "Europe"},
		{ID: 50, Name: "UEFA Euro", Country: "Europe"},
		{ID: 292, Name: "UEFA Women's Euro", Country: "Europe"},
		{ID: 9375, Name: "Women's UEFA Champions League", Country: "Europe"},
		// Domestic Cups
		{ID: 138, Name: "Copa del Rey", Country: "Spain"},
		{ID: 139, Name: "Supercopa de España", Country: "Spain"},
		{ID: 132, Name: "FA Cup", Country: "England"},
		{ID: 209, Name: "DFB Pokal", Country: "Germany"},
		{ID: 10650, Name: "Women's DFB Pokal", Country: "Germany"},
		{ID: 141, Name: "Coppa Italia", Country: "Italy"},
		{ID: 134, Name: "Coupe de France", Country: "France"},
		//Portugal Cups
		{ID: 186, Name: "Taça de Portugal", Country: "Portugal"},
		{ID: 187, Name: "Taça da Liga", Country: "Portugal"},
		{ID: 188, Name: "Supertaça Cândido de Oliveira", Country: "Portugal"},
	},
	RegionAmerica: {
		// North America Competitions
		{ID: 297, Name: "CONCACAF Champions Cup", Country: "North America"},
		{ID: 298, Name: "CONCACAF Gold Cup", Country: "North America"},
		{ID: 9821, Name: "CONCACAF Nations League", Country: "North America"},
		{ID: 10043, Name: "Leagues Cup", Country: "North America"},
		// South America
		{ID: 268, Name: "Brasileirão Série A", Country: "Brazil"},
		{ID: 8814, Name: "Brasileirão Série B", Country: "Brazil"},
		{ID: 9067, Name: "Copa do Brasil", Country: "Brazil"},
		{ID: 10077, Name: "Supercopa do Brasil", Country: "Brazil"},
		{ID: 10244, Name: "Paulista", Country: "Brazil"},
		{ID: 10272, Name: "Carioca", Country: "Brazil"},
		{ID: 10273, Name: "Mineiro", Country: "Brazil"},
		{ID: 10274, Name: "Gaúcho", Country: "Brazil"},
		{ID: 9429, Name: "Nordeste", Country: "Brazil"},
		{ID: 10291, Name: "Goiano", Country: "Brazil"},
		{ID: 44, Name: "Copa America", Country: "South America"},
		{ID: 9490, Name: "Copa Colombia", Country: "Colombia"},
		{ID: 45, Name: "Copa Libertadores", Country: "South America"},
		{ID: 299, Name: "Copa Sudamericana", Country: "South America"},
		{ID: 491, Name: "Recopa Sudamericana", Country: "South America"},
		{ID: 112, Name: "Liga Profesional", Country: "Argentina"},
		{ID: 274, Name: "Primera A", Country: "Colombia"},
		{ID: 9125, Name: "Primera B", Country: "Colombia"},
		{ID: 161, Name: "Primera Division", Country: "Uruguay"},
		{ID: 273, Name: "Primera Division", Country: "Chile"},
		{ID: 131, Name: "Liga 1", Country: "Peru"},
		{ID: 246, Name: "Serie A", Country: "Ecuador"},
		// North America
		{ID: 130, Name: "MLS", Country: "USA"},
		{ID: 8972, Name: "USL Championship", Country: "USA"},
		{ID: 9296, Name: "USL League One", Country: "USA"},
		{ID: 9134, Name: "NWSL", Country: "USA"},
		{ID: 10699, Name: "USL Gainsbridge Super League", Country: "USA"},
		{ID: 230, Name: "Liga MX", Country: "Mexico"},
	},
	RegionGlobal: {
		// Middle East
		{ID: 536, Name: "Saudi Pro League", Country: "Saudi Arabia"},
		// Asia
		{ID: 525, Name: "AFC Champions League Elite", Country: "Asia"},
		{ID: 9469, Name: "AFC Champions League Two", Country: "Asia"},
		{ID: 9478, Name: "Indian Super League", Country: "India"},
		{ID: 223, Name: "J. League", Country: "Japan"},
		{ID: 9080, Name: "K League 1", Country: "South Korea"},
		{ID: 9137, Name: "Chinese League One", Country: "China"},
		{ID: 535, Name: "Qatar Stars League", Country: "Qatar"},
		// Oceania
		{ID: 113, Name: "A-League", Country: "Australia"},
		// Africa
		{ID: 526, Name: "CAF Champions League", Country: "Africa"},
		{ID: 519, Name: "Egyptian Premier League", Country: "Egypt"},
		{ID: 537, Name: "Premier Soccer League", Country: "South Africa"},
		{ID: 530, Name: "Botola Pro", Country: "Morocco"},
		// International Competitions
		{ID: 289, Name: "Africa Cup of Nations", Country: "International"},
		{ID: 77, Name: "FIFA World Cup", Country: "International"},
		{ID: 76, Name: "Women's FIFA World Cup", Country: "International"},
		{ID: 78, Name: "FIFA Club World Cup", Country: "International"},
		{ID: 9806, Name: "UEFA Nations League", Country: "International"},
		{ID: 10304, Name: "Finalissima", Country: "International"},
		{ID: 489, Name: "Club Friendlies", Country: "International"},
		{ID: 114, Name: "International Friendlies", Country: "International"},
	},
}

// Settings represents user preferences stored in settings.yaml.
type Settings struct {
	// SelectedLeagues contains the IDs of leagues the user wants to follow.
	// If empty, all supported leagues are used.
	SelectedLeagues []int `yaml:"selected_leagues"`
}

// SettingsPath returns the path to the settings file.
func SettingsPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsFileName), nil
}

// LoadSettings reads settings from the settings.yaml file.
// Returns default settings (empty selection = all leagues) if file doesn't exist.
func LoadSettings() (*Settings, error) {
	path, err := SettingsPath()
	if err != nil {
		return &Settings{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No settings file - return empty settings (will use all leagues)
			return &Settings{}, nil
		}
		return &Settings{}, err
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		// Invalid YAML - return empty settings
		return &Settings{}, nil
	}

	return &settings, nil
}

// SaveSettings writes settings to the settings.yaml file atomically.
// The payload is marshaled to a temporary file in the same directory and
// renamed into place, so readers never observe a half-written file and a
// failed save leaves the previous settings untouched.
func SaveSettings(settings *Settings) error {
	path, err := SettingsPath()
	if err != nil {
		return err
	}

	// Make sure the config directory exists (ConfigDir normally creates it,
	// but do not rely on that when the directory was removed out from under us).
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".settings.yaml.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// os.Rename is atomic on POSIX and replaces the destination on Windows
	// (Go uses MoveFileEx with replace semantics).
	return os.Rename(tmpPath, path)
}

// DefaultLeagueIDs contains the default leagues used when no selection is made.
// These are the most popular leagues for efficient API usage.
var DefaultLeagueIDs = []int{
	47, // Premier League
	87, // La Liga
	42, // UEFA Champions League
	77, // FIFA World Cup
}

// NormalizeLeagueIDs canonicalizes a league selection into the stable snapshot
// used for an entire query round:
//   - an empty selection falls back to DefaultLeagueIDs;
//   - duplicate IDs are collapsed;
//   - IDs are sorted so identical selections always produce identical
//     snapshots (and therefore identical cache keys), regardless of the order
//     in which the user toggled them.
//
// The returned slice is always a fresh copy that callers may retain safely.
func NormalizeLeagueIDs(ids []int) []int {
	if len(ids) == 0 {
		fallback := append([]int(nil), DefaultLeagueIDs...)
		slices.Sort(fallback)
		return fallback
	}

	seen := make(map[int]struct{}, len(ids))
	normalized := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	slices.Sort(normalized)
	return normalized
}

// LoadActiveLeagueIDs loads the current settings once and returns the
// normalized league snapshot to use for a query round. Callers that fan work
// out into batches must call this once and thread the returned snapshot
// through every batch instead of re-reading settings per batch.
// Load errors are returned to the caller; use ActiveLeagueIDs for the
// default-fallback convenience path.
func LoadActiveLeagueIDs() ([]int, error) {
	settings, err := LoadSettings()
	if err != nil {
		return NormalizeLeagueIDs(nil), err
	}
	return NormalizeLeagueIDs(settings.SelectedLeagues), nil
}

// ActiveLeagueIDs returns the league IDs that should be used for API calls.
// If no leagues are selected in settings (or settings cannot be loaded),
// returns the default leagues (not all). The result is normalized: sorted,
// deduplicated, and safe to retain as a per-query snapshot.
func ActiveLeagueIDs() []int {
	ids, _ := LoadActiveLeagueIDs()
	return ids
}

// AllLeagueIDs returns all supported league IDs (used as fallback).
func AllLeagueIDs() []int {
	totalLeagues := 0
	for _, leagues := range AllSupportedLeagues {
		totalLeagues += len(leagues)
	}

	ids := make([]int, 0, totalLeagues)
	for _, leagues := range AllSupportedLeagues {
		for _, league := range leagues {
			ids = append(ids, league.ID)
		}
	}
	return ids
}

// IsLeagueSelected checks if a league ID is in the selected list.
func (s *Settings) IsLeagueSelected(leagueID int) bool {
	return slices.Contains(s.SelectedLeagues, leagueID)
}

// GetAllRegions returns a list of all available regions in order.
func GetAllRegions() []string {
	return []string{RegionEurope, RegionAmerica, RegionGlobal}
}

// GetLeaguesForRegion returns all leagues for a specific region.
func GetLeaguesForRegion(region string) []LeagueInfo {
	return AllSupportedLeagues[region]
}
