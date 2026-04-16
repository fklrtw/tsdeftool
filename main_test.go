package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		distance int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "ac", 1},
		{"abc", "bc", 1},
		{"kitten", "sitting", 3},
		{"flaw", "lawn", 2},
	}

	for _, tt := range tests {
		got := levenshteinDistance(tt.s1, tt.s2)
		if got != tt.distance {
			t.Errorf("levenshteinDistance(%q, %q) = %d; want %d", tt.s1, tt.s2, got, tt.distance)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		vals []int
		want int
	}{
		{[]int{1, 2, 3}, 1},
		{[]int{3, 2, 1}, 1},
		{[]int{2, 1, 3}, 1},
		{[]int{1}, 1},
		{[]int{-1, 0, 1}, -1},
	}

	for _, tt := range tests {
		got := min(tt.vals...)
		if got != tt.want {
			t.Errorf("min(%v) = %d; want %d", tt.vals, got, tt.want)
		}
	}
}

func TestAssertDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Case 1: Directory already exists
	path1 := filepath.Join(tmpDir, "shadow_realm")
	if err := os.Mkdir(path1, 0755); err != nil {
		t.Fatal(err)
	}
	p1 := path1
	err := assertDirectory(&p1, "test1", false)
	if err != nil {
		t.Errorf("assertDirectory(exist) failed: %v", err)
	}

	// Case 2: Directory doesn't exist, create it
	path2 := filepath.Join(tmpDir, "dragon_lair")
	p2 := path2
	err = assertDirectory(&p2, "test2", true)
	if err != nil {
		t.Errorf("assertDirectory(new, create=true) failed: %v", err)
	}
	if _, err := os.Stat(path2); os.IsNotExist(err) {
		t.Errorf("assertDirectory(new, create=true) did not create directory")
	}

	// Case 3: Directory doesn't exist, don't create it
	path3 := filepath.Join(tmpDir, "missing_artifact")
	p3 := path3
	err = assertDirectory(&p3, "test3", false)
	if err == nil {
		t.Errorf("assertDirectory(missing, create=false) should have failed")
	}

	// Case 4: Not a directory
	path4 := filepath.Join(tmpDir, "magic_scroll.txt")
	if err := os.WriteFile(path4, []byte("abracadabra"), 0644); err != nil {
		t.Fatal(err)
	}
	p4 := path4
	err = assertDirectory(&p4, "test4", false)
	if err == nil {
		t.Errorf("assertDirectory(file, create=false) should have failed")
	}
}

func TestWriteCSV(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "inventory.csv")
	rows := [][]string{
		{"potion", "count"},
		{"mana_regen", "5"},
	}

	err := writeCSV(csvPath, rows)
	if err != nil {
		t.Fatalf("writeCSV failed: %v", err)
	}

	content, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("reading CSV failed: %v", err)
	}

	expected := "potion;count\nmana_regen;5"
	if string(content) != expected {
		t.Errorf("writeCSV content = %q; want %q", string(content), expected)
	}
}

func TestWriteCompanyFile(t *testing.T) {
	tmpDir := t.TempDir()
	companyName := "elven_express"
	vType := "truck"
	vehicles := []string{"silver_steed", "golden_chariot"}

	err := writeCompanyFile(companyName, vType, vehicles, tmpDir)
	if err != nil {
		t.Fatalf("writeCompanyFile failed: %v", err)
	}

	fileName := "traffic.truck_elven_express.sii"
	filePath := filepath.Join(tmpDir, fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading company file failed: %v", err)
	}

	// Note: writeCompanyFile sorts the vehicles
	expectedPrefix := "SiiNunit\n{\n\ncountry_traffic_info : .country.info.traffic.golden_chariot {\n    object: traffic.golden_chariot\n    spawn_frequency : 0.00\n}\n"
	if !strings.HasPrefix(string(content), expectedPrefix) {
		t.Errorf("writeCompanyFile content unexpected:\n%s", string(content))
	}
}

func TestReadSourceFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a dummy .sii file
	siiContent := `traffic_vehicle : traffic.iron_golem // forge_masters
traffic_trailer : traffic.wooden_cart // plain
`
	err := os.WriteFile(filepath.Join(tmpDir, "test.sii"), []byte(siiContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	companyMap := make(map[string]*Company)
	err = readSourceFiles(tmpDir, companyMap)
	if err != nil {
		t.Fatalf("readSourceFiles failed: %v", err)
	}

	if _, ok := companyMap["forge_masters"]; !ok {
		t.Error("company 'forge_masters' not found in map")
	} else {
		if len(companyMap["forge_masters"].Trucks) != 1 {
			t.Errorf("forge_masters trucks count = %d; want 1", len(companyMap["forge_masters"].Trucks))
		}
		if companyMap["forge_masters"].Trucks[0].Name != "iron_golem" {
			t.Errorf("forge_masters truck name = %q; want 'iron_golem'", companyMap["forge_masters"].Trucks[0].Name)
		}
	}

	if _, ok := companyMap["plain"]; !ok {
		t.Error("company 'plain' not found in map")
	} else {
		if len(companyMap["plain"].Trailers) != 1 {
			t.Errorf("plain trailers count = %d; want 1", len(companyMap["plain"].Trailers))
		}
		if companyMap["plain"].Trailers[0].Name != "wooden_cart" {
			t.Errorf("plain trailer name = %q; want 'wooden_cart'", companyMap["plain"].Trailers[0].Name)
		}
	}
}

func TestNoDuplicateVehicles(t *testing.T) {
	t.Run("CleanData", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Two different files, unique identifiers within the same company
		f1 := `traffic_vehicle : traffic.gryphon_rider // citadel_guard
traffic_trailer : traffic.supply_wagon // citadel_guard`
		f2 := `traffic_vehicle : traffic.pegasus_knight // citadel_guard
traffic_trailer : traffic.ballista // citadel_guard`

		if err := os.WriteFile(filepath.Join(tmpDir, "f1.sii"), []byte(f1), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "f2.sii"), []byte(f2), 0644); err != nil {
			t.Fatal(err)
		}

		companyMap := make(map[string]*Company)
		if err := readSourceFiles(tmpDir, companyMap); err != nil {
			t.Fatal(err)
		}

		checkDuplicates(t, companyMap)
	})

	t.Run("DetectDuplicates", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Duplicate truck identifier in different files for the same company
		f1 := `traffic_vehicle : traffic.cursed_spirit // graveyard_shift`
		f2 := `traffic_vehicle : traffic.cursed_spirit // graveyard_shift`

		if err := os.WriteFile(filepath.Join(tmpDir, "f1.sii"), []byte(f1), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "f2.sii"), []byte(f2), 0644); err != nil {
			t.Fatal(err)
		}

		companyMap := make(map[string]*Company)
		if err := readSourceFiles(tmpDir, companyMap); err != nil {
			t.Fatal(err)
		}

		found := false
		for _, company := range companyMap {
			seen := make(map[string]bool)
			for _, v := range company.Trucks {
				if seen[v.Name] {
					found = true
					break
				}
				seen[v.Name] = true
			}
		}
		if !found {
			t.Error("Expected to find a duplicate truck, but none were detected")
		}
	})
}

func TestRunIntegration(t *testing.T) {
	tmpSource := t.TempDir()
	tmpDest := t.TempDir()

	// Create some dummy data in source
	siiContent := `traffic_vehicle : traffic.spectral_steed // mystic_guild`
	err := os.WriteFile(filepath.Join(tmpSource, "data.sii"), []byte(siiContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test successful run
	args := []string{
		"-source-directory", tmpSource,
		"-destination-directory", tmpDest,
		"-log-level", "debug",
	}

	err = run(args)
	if err != nil {
		t.Errorf("run() failed: %v", err)
	}

	// Verify output
	expectedFile := filepath.Join(tmpDest, "traffic.truck_mystic_guild.sii")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected output file %s not found", expectedFile)
	}

	// Test help command
	err = run([]string{"help"})
	if err != nil {
		t.Errorf("run(help) failed: %v", err)
	}

	// Test version command
	err = run([]string{"version"})
	if err != nil {
		t.Errorf("run(version) failed: %v", err)
	}
}

// checkDuplicates is a helper to verify no duplicates exist in the map
func checkDuplicates(t *testing.T, companyMap map[string]*Company) {
	for companyName, company := range companyMap {
		// Check Trucks
		seenTrucks := make(map[string]string)
		for _, v := range company.Trucks {
			if firstFile, dup := seenTrucks[v.Name]; dup {
				t.Errorf("Duplicate truck identifier %q for company %q found in %q (first seen in %q)",
					v.Name, companyName, v.FileName, firstFile)
			}
			seenTrucks[v.Name] = v.FileName
		}

		// Check Trailers
		seenTrailers := make(map[string]string)
		for _, v := range company.Trailers {
			if firstFile, dup := seenTrailers[v.Name]; dup {
				t.Errorf("Duplicate trailer identifier %q for company %q found in %q (first seen in %q)",
					v.Name, companyName, v.FileName, firstFile)
			}
			seenTrailers[v.Name] = v.FileName
		}
	}
}
