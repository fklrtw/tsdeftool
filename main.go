// Package main provides a utility for American Truck Simulator (ATS) and Euro Truck Simulator 2 (ETS2) modders.
// It parses traffic vehicle and trailer definitions from .sii and .sui files,
// categorizes them by company (based on comments in the source files),
// and generates consolidated traffic definition files for use in map mods or traffic density mods.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// PrettyHandler is a custom slog.Handler that formats logs for the CLI.
type PrettyHandler struct {
	opts slog.HandlerOptions
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String()
	if r.Level == slog.LevelDebug {
		level = "DEBUG"
	}

	fmt.Printf("[%s] %s", level, r.Message)

	r.Attrs(func(a slog.Attr) bool {
		fmt.Printf(" %s=%v", a.Key, a.Value)
		return true
	})

	fmt.Println()
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

// Vehicle represents a vehicle (trailer or truck) found in the source files.
type Vehicle struct {
	FileName string // The source file path where the vehicle was found.
	Name     string // The internal name/identifier of the vehicle.
}

// Company represents a company and its associated lists of trailers and trucks.
type Company struct {
	Trailers []Vehicle
	Trucks   []Vehicle
}

// VehicleType is a custom string type representing either "trailer" or "truck".
type VehicleType string

const (
	// Trailer represents a trailer vehicle type.
	Trailer VehicleType = "trailer"
	// Truck represents a truck vehicle type.
	Truck VehicleType = "truck"
)

// vehicleTypes lists the supported vehicle types for iteration.
var vehicleTypes = []VehicleType{Trailer, Truck}

// vehicleRegex is used to identify vehicle or trailer definitions in source files.
// It matches lines starting with 'traffic_vehicle' or 'traffic_trailer'.
var vehicleRegex = regexp.MustCompile(`^traffic_(vehicle|trailer)\s+:\s+traffic\.(\S+)(\s+\/\/\s+(\S+))?`)

func main() {
	// CLI Flags
	sourceDir := flag.String("source-directory", ".", "Source directory/where the source files are")
	destDir := flag.String("destination-directory", "output", "Destination directory/where the generated files should go")
	maxDistance := flag.Int("maximum-levenshtein-distance", 2, "(Maximum) Levenshtein distance of two company names to be considered a typo")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	logLevelStr := flag.String("log-level", "warn", "Log level (debug, info, warn, error)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [command]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  help     Print this help message\n")
		fmt.Fprintf(os.Stderr, "  version  Print version and exit\n")
	}

	flag.Parse()

	// Configure slog based on the log-level flag
	var level slog.Level
	switch strings.ToLower(*logLevelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelWarn
	}

	logger := slog.New(&PrettyHandler{
		opts: slog.HandlerOptions{Level: level},
	})
	slog.SetDefault(logger)

	if *versionFlag || (flag.NArg() > 0 && flag.Arg(0) == "version") {
		fmt.Println("0.0.2")
		return
	}

	if flag.NArg() > 0 && flag.Arg(0) == "help" {
		flag.Usage()
		return
	}

	start := time.Now()

	// Ensure source and destination directories exist
	err := assertDirectory(sourceDir, "source", true)
	if err != nil {
		slog.Error("Directory assertion failed", "error", err)
		os.Exit(1)
	}

	err = assertDirectory(destDir, "destination", true)
	if err != nil {
		slog.Error("Directory assertion failed", "error", err)
		os.Exit(1)
	}

	companyMap := make(map[string]*Company)

	// Step 1: Read all source files and populate the company map
	fmt.Printf("[LOG] Recursively reading source files in '%s'...\n", *sourceDir)
	err = readSourceFiles(*sourceDir, companyMap)
	if err != nil {
		slog.Error("Failed to read source files", "error", err)
		os.Exit(1)
	}

	// Step 2: Sort company names for deterministic output
	fmt.Printf("[LOG] Processing %d companies, writing to '%s'...\n", len(companyMap), *destDir)
	companyNames := make([]string, 0, len(companyMap))
	for name := range companyMap {
		companyNames = append(companyNames, name)
	}
	sort.Strings(companyNames)

	// Step 3: Generate company-specific .sii files and prepare glossary
	glossary := [][]string{
		{"company", "type", "vehicle", "file"},
	}

	for _, companyName := range companyNames {
		company := companyMap[companyName]
		for _, vType := range vehicleTypes {
			var vehicles []Vehicle
			if vType == Trailer {
				vehicles = company.Trailers
			} else {
				vehicles = company.Trucks
			}

			if len(vehicles) > 0 {
				vehicleNames := make([]string, len(vehicles))
				for i, v := range vehicles {
					vehicleNames[i] = v.Name
				}
				err := writeCompanyFile(companyName, string(vType), vehicleNames, *destDir)
				if err != nil {
					slog.Error("Failed to write company file", "company", companyName, "type", vType, "error", err)
					continue
				}

				// Sort vehicles by name for glossary consistency
				sort.Slice(vehicles, func(i, j int) bool {
					return vehicles[i].Name < vehicles[j].Name
				})

				for _, v := range vehicles {
					glossary = append(glossary, []string{companyName, string(vType), v.Name, v.FileName})
				}
			}
		}
	}

	// Step 4: Detect potential typos in company names using Levenshtein distance
	companyTypos := [][]string{
		{"company A", "company B", "distance"},
	}

	for i := 0; i < len(companyNames); i++ {
		companyA := companyNames[i]
		for j := i + 1; j < len(companyNames); j++ {
			companyB := companyNames[j]
			distance := levenshteinDistance(companyA, companyB)
			if distance <= *maxDistance {
				companyTypos = append(companyTypos, []string{
					companyA,
					companyB,
					fmt.Sprintf("%d", distance),
				})
			}
		}
	}

	// Step 5: Write metadata files
	err = writeCSV(filepath.Join(*destDir, "_typos.csv"), companyTypos)
	if err != nil {
		slog.Error("Failed to write typos CSV", "error", err)
	}

	err = writeCSV(filepath.Join(*destDir, "_glossary.csv"), glossary)
	if err != nil {
		slog.Error("Failed to write glossary CSV", "error", err)
	}

	fmt.Printf("ok: %v\n", time.Since(start))
}

// assertDirectory checks if a directory exists and optionally creates it.
func assertDirectory(directory *string, identifier string, create bool) error {
	info, err := os.Lstat(*directory)
	if err != nil {
		if os.IsNotExist(err) && create {
			slog.Info("Creating directory", "identifier", identifier, "path", *directory)
			return os.MkdirAll(*directory, 0755)
		}
		return fmt.Errorf("provided %s directory '%s' does not exist", identifier, *directory)
	}

	if !info.IsDir() {
		return fmt.Errorf("provided %s directory '%s' is not a directory", identifier, *directory)
	}

	absPath, _ := filepath.Abs(*directory)

	*directory = absPath

	return nil
}

// readSourceFiles recursively reads all .sui and .sii files in a directory and populates companyMap.
func readSourceFiles(directory string, companyMap map[string]*Company) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			err = readSourceFiles(path, companyMap)
			if err != nil {
				return err
			}
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".sui" && ext != ".sii" {
			slog.Debug("Ignoring file", "path", path, "reason", "wrong extension")
			continue
		}

		slog.Info("Reading file", "path", path)
		file, err := os.Open(path)
		if err != nil {
			slog.Warn("Could not open file", "path", path, "error", err)
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			match := vehicleRegex.FindStringSubmatch(line)
			if match == nil {
				continue
			}

			vType := Truck
			if match[1] == "trailer" {
				vType = Trailer
			}

			name := match[2]
			company := "plain"
			if len(match) > 4 && match[4] != "" {
				company = strings.ToLower(strings.Replace(match[4], "/", "_", 1))
				company = strings.TrimSpace(company)
			}

			if _, ok := companyMap[company]; !ok {
				slog.Info("Adding new company", "company", company)
				companyMap[company] = &Company{
					Trailers: []Vehicle{},
					Trucks:   []Vehicle{},
				}
			}

			slog.Debug("Adding vehicle to company", "type", vType, "name", name, "company", company)
			vehicle := Vehicle{FileName: path, Name: name}
			if vType == Truck {
				companyMap[company].Trucks = append(companyMap[company].Trucks, vehicle)
			} else {
				companyMap[company].Trailers = append(companyMap[company].Trailers, vehicle)
			}
		}
		file.Close()

		if err := scanner.Err(); err != nil {
			slog.Warn("Error scanning file", "path", path, "error", err)
		}
	}

	return nil
}

// writeCompanyFile generates a .sii file containing country_traffic_info for a specific company and vehicle type.
func writeCompanyFile(companyName string, vehicleType string, vehicles []string, destinationDirectory string) error {
	fileName := fmt.Sprintf("traffic.%s_%s.sii", vehicleType, companyName)

	slog.Info("Generating company file", "file", fileName, "company", companyName, "vehicle_count", len(vehicles))

	var sb strings.Builder
	sb.WriteString("SiiNunit\n{\n")

	slices.Sort(vehicles)

	for _, vehicle := range vehicles {
		sb.WriteString(fmt.Sprintf("\ncountry_traffic_info : .country.info.traffic.%s {\n", vehicle))
		sb.WriteString(fmt.Sprintf("    object: traffic.%s\n", vehicle))
		sb.WriteString("    spawn_frequency : 0.00\n}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(filepath.Join(destinationDirectory, fileName), []byte(sb.String()), 0644)
}

// writeCSV writes a 2D string slice to a file in semicolon-separated CSV format.
func writeCSV(path string, rows [][]string) error {
	var sb strings.Builder
	for i, row := range rows {
		sb.WriteString(strings.Join(row, ";"))
		if i < len(rows)-1 {
			sb.WriteString("\n")
		}
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// levenshteinDistance calculates the Levenshtein distance between two strings to identify similar company names.
func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	m := len(r1)
	n := len(r2)
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
	}
	for i := 0; i <= m; i++ {
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}
	for j := 1; j <= n; j++ {
		for i := 1; i <= m; i++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}
	return d[m][n]
}

// min returns the minimum value from a list of integers.
func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
