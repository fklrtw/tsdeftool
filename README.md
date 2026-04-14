# Truck Simulator Definition Tool

A high-performance utility for American Truck Simulator (ATS) and Euro Truck Simulator 2 (ETS2) modders. This tool automates the process of organizing and generating traffic definition files based on company-specific comments in source files.

## What it does

In SCS Prism3D engine games (ATS/ETS2), AI traffic is defined using `.sii` and `.sui` files. Often, modders want to group specific trucks or trailers by the company they represent (e.g., Amazon, FedEx, Walmart) to control their appearance or frequency in specific regions.

This tool:
1.  **Parses Source Files**: Recursively reads through all `.sii` and `.sui` files in a source directory.
2.  **Identifies Vehicles/Trailers**: Looks for `traffic_vehicle` and `traffic_trailer` definitions.
3.  **Extracts Company Context**: Uses the inline comment at the end of a definition line (e.g., `traffic_vehicle : traffic.680ng_0017 // Amazon`) to determine which company a vehicle belongs to.
4.  **Generates Traffic Definitions**: Creates consolidated `.sii` files for each company (e.g., `traffic.truck_amazon.sii`) containing `country_traffic_info` units with a default `spawn_frequency` of `0.00`.
5.  **Metadata Generation**:
    *   `_glossary.csv`: A complete list of all found vehicles, their type, company, and source file path.
    *   `_typos.csv`: Uses Levenshtein distance to flag potential typos in company names (e.g., identifying "Amazon" and "Amzon" as likely the same company).

## Usage

The easiest way to use the tool is to download the latest pre-built executable for your operating system.

### Download
You can find the latest releases for Windows and Linux here:
**[👉 Download Latest Releases](https://github.com/fklrtw/tsdeftool/releases)**

### Running the tool
Run the executable from your terminal or command prompt:

**Windows:**
```powershell
.\tsdeftool-windows.exe [options]
```

**Linux:**
```bash
./tsdeftool-linux [options]
```

### Options
- `-source-directory`: Where the `.sui`/`.sii` files are located (default: `.`)
- `-destination-directory`: Where the generated files should go (default: `output`)
- `-maximum-levenshtein-distance`: Sensitivity for typo detection (default: `2`)
- `-log-level`: Control verbosity (`debug`, `info`, `warn`, `error`) (default: `info`)
- `-version`: Print version and exit
- `help`: (Command) Print usage information

### Example
To parse files in a `defs` folder and output to `generated_defs` with high verbosity:
```bash
./tsdeftool-linux -source-directory ./defs -destination-directory ./generated_defs -log-level debug
```

## Building from Source

If you prefer to build the tool yourself, you will need **Go 1.21** or later installed.

### Build Commands
**Linux:**
```bash
GOOS=linux GOARCH=amd64 go build -o tsdeftool-linux main.go
```

**Windows:**
```bash
GOOS=windows GOARCH=amd64 go build -o tsdeftool-windows.exe main.go
```

## Internal Formats
The generated files use the standard `SiiNunit` wrapper required by the SCS engine:
```sii
SiiNunit
{
country_traffic_info : .country.info.traffic.vehicle_name {
    object: traffic.vehicle_name
    spawn_frequency : 0.00
}
}
```
This allows modders to easily include these files in their mods and adjust frequencies as needed for specific countries or states.
