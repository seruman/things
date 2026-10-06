package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realMain(
		ctx,
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(
	ctx context.Context,
	stdout io.Writer,
	stderr io.Writer,
	osargs []string,
) error {
	exec := osargs[0]
	client := NewClient()

	stationsFS := flag.NewFlagSet("stations", flag.ContinueOnError)
	stationsFS.SetOutput(stderr)
	var (
		flagStationsBrand     string
		flagStationsNear      string
		flagStationsKM        float64
		flagStationsAvailable bool
		flagStationsJSON      bool
	)
	stationsFS.StringVar(&flagStationsBrand, "brand", "", "brand slug to filter by, zes for example")
	stationsFS.StringVar(&flagStationsNear, "near", "", "lat,lng to filter and sort by distance")
	stationsFS.Float64Var(&flagStationsKM, "km", 5, "radius in km around -near")
	stationsFS.BoolVar(&flagStationsAvailable, "available", false, "only stations with a free socket")
	stationsFS.BoolVar(&flagStationsJSON, "json", false, "output JSON")

	stationsCmd := &ffcli.Command{
		Name:       "stations",
		ShortUsage: fmt.Sprintf("%s stations [flags]", exec),
		ShortHelp:  "List stations with live availability (app backend).",
		FlagSet:    stationsFS,
		Exec: func(ctx context.Context, _ []string) error {
			stations, err := client.Stations(ctx)
			if err != nil {
				return err
			}

			if flagStationsBrand != "" {
				stations = filter(stations, func(s StationSummary) bool {
					return strings.EqualFold(s.Brand, flagStationsBrand)
				})
			}

			if flagStationsAvailable {
				stations = filter(stations, func(s StationSummary) bool { return s.Available })
			}

			near := flagStationsNear != ""
			var lat, lng float64
			if near {
				lat, lng, err = parseLatLng(flagStationsNear)
				if err != nil {
					return fmt.Errorf("-near: %w", err)
				}

				dist := func(s StationSummary) float64 { return distanceKM(lat, lng, s.Lat, s.Lng) }
				stations = filter(stations, func(s StationSummary) bool { return dist(s) <= flagStationsKM })
				slices.SortFunc(stations, func(a, b StationSummary) int {
					return cmpFloat(dist(a), dist(b))
				})
			}

			if flagStationsJSON {
				return writeJSON(stdout, stations)
			}

			headers := []string{"ID", "Brand", "Title", "Sockets", "Available"}
			if near {
				headers = append(headers, "Km")
			}

			table := newTable(stdout, headers...)
			for _, s := range stations {
				row := []string{
					strconv.FormatInt(s.ID, 10),
					s.Brand,
					s.Title,
					strconv.Itoa(len(s.Sockets)),
					yesno(s.Available),
				}
				if near {
					row = append(row, fmt.Sprintf("%.2f", distanceKM(lat, lng, s.Lat, s.Lng)))
				}
				table.Append(row)
			}
			table.Render()
			return nil
		},
	}

	stationFS := flag.NewFlagSet("station", flag.ContinueOnError)
	stationFS.SetOutput(stderr)
	var (
		flagStationAt   string
		flagStationJSON bool
	)
	stationFS.StringVar(&flagStationAt, "at", "", `report status as of this Istanbul time, YYYY-MM-DD or YYYY-MM-DD HH:MM, now by default`)
	stationFS.BoolVar(&flagStationJSON, "json", false, "output JSON")

	stationCmd := &ffcli.Command{
		Name:       "station",
		ShortUsage: fmt.Sprintf("%s station [flags] <id>", exec),
		ShortHelp:  "Show a station's sockets, prices and live status (app backend).",
		FlagSet:    stationFS,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return flag.ErrHelp
			}

			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("station id: %w", err)
			}

			at := time.Now()
			if flagStationAt != "" {
				at, err = parseAt(flagStationAt)
				if err != nil {
					return fmt.Errorf("-at: %w", err)
				}
			}

			station, err := client.Station(ctx, id, at)
			if err != nil {
				return err
			}

			if flagStationJSON {
				return writeJSON(stdout, station)
			}

			fmt.Fprintf(stdout, "%s (%d)\n", station.Title, station.ID)
			fmt.Fprintf(stdout, "%s\n", station.Address)
			fmt.Fprintf(stdout, "%s / %s / green=%s / %s\n", station.Brand, station.OperatorTitle, yesno(station.Green.Bool()), at.Format("2006-01-02 15:04"))
			fmt.Fprintln(stdout)

			table := newTable(stdout, "Socket", "Type", "kW", "₺/kWh", "Status", "Until")
			for _, s := range station.Sockets {
				w, _ := s.StatusAt(at)
				until := ""
				if !w.EndTime.IsZero() {
					until = w.EndTime.Format("15:04")
				}
				table.Append([]string{
					s.SocketNumber,
					s.SubType,
					strconv.FormatFloat(s.Power, 'f', -1, 64),
					fmt.Sprintf("%.2f", s.Price),
					string(w.Status),
					until,
				})
			}
			table.Render()
			return nil
		},
	}

	registryFS := flag.NewFlagSet("registry", flag.ContinueOnError)
	registryFS.SetOutput(stderr)
	var (
		flagRegistrySource string
		flagRegistryBrand  string
		flagRegistryNear   string
		flagRegistryKM     float64
		flagRegistryJSON   bool
	)
	registryFS.StringVar(&flagRegistrySource, "source", "mirror", "mirror or gateway; mirror is prizy-data's nightly copy with fewer fields, gateway is EPDK itself with an hourly quota")
	registryFS.StringVar(&flagRegistryBrand, "brand", "", "brand slug to filter by, zes for example")
	registryFS.StringVar(&flagRegistryNear, "near", "", "lat,lng to filter and sort by distance")
	registryFS.Float64Var(&flagRegistryKM, "km", 5, "radius in km around -near")
	registryFS.BoolVar(&flagRegistryJSON, "json", false, "output JSON")

	registryCmd := &ffcli.Command{
		Name:       "registry",
		ShortUsage: fmt.Sprintf("%s registry [flags]", exec),
		ShortHelp:  "List stations as filed with EPDK: rated power, operator, licence. No live data.",
		FlagSet:    registryFS,
		Exec: func(ctx context.Context, _ []string) error {
			var rows []registryRow
			var raw any
			switch flagRegistrySource {
			case "mirror":
				mirror, err := client.Mirror(ctx)
				if err != nil {
					return err
				}
				rows = mapz(mirror.Stations, mirrorRow)
				raw = mirror.Stations
			case "gateway":
				registry, err := client.Registry(ctx)
				if err != nil {
					return err
				}
				rows = mapz(registry.Data, gatewayRow)
				raw = registry.Data
			default:
				return fmt.Errorf("-source: expected mirror or gateway, got %q", flagRegistrySource)
			}

			if flagRegistryBrand != "" {
				rows = filter(rows, func(r registryRow) bool {
					return strings.EqualFold(r.brand, flagRegistryBrand)
				})
			}

			near := flagRegistryNear != ""
			var lat, lng float64
			var err error
			if near {
				lat, lng, err = parseLatLng(flagRegistryNear)
				if err != nil {
					return fmt.Errorf("-near: %w", err)
				}

				dist := func(r registryRow) float64 { return distanceKM(lat, lng, r.lat, r.lng) }
				rows = filter(rows, func(r registryRow) bool { return dist(r) <= flagRegistryKM })
				slices.SortFunc(rows, func(a, b registryRow) int {
					return cmpFloat(dist(a), dist(b))
				})
			}

			if flagRegistryJSON {
				if flagRegistryBrand == "" && !near {
					return writeJSON(stdout, raw)
				}
				return writeJSON(stdout, mapz(rows, func(r registryRow) any { return r.raw }))
			}

			headers := []string{"No", "Brand", "Name", "Sockets", "Max kW", "Operator"}
			if near {
				headers = append(headers, "Km")
			}

			table := newTable(stdout, headers...)
			for _, r := range rows {
				row := []string{
					r.no,
					r.brand,
					r.name,
					strconv.Itoa(r.sockets),
					strconv.FormatFloat(r.maxKW, 'f', -1, 64),
					r.operator,
				}
				if near {
					row = append(row, fmt.Sprintf("%.2f", distanceKM(lat, lng, r.lat, r.lng)))
				}
				table.Append(row)
			}
			table.Render()
			return nil
		},
	}

	citiesFS := flag.NewFlagSet("cities", flag.ContinueOnError)
	citiesFS.SetOutput(stderr)
	var flagCitiesJSON bool
	citiesFS.BoolVar(&flagCitiesJSON, "json", false, "output JSON")

	citiesCmd := &ffcli.Command{
		Name:       "cities",
		ShortUsage: fmt.Sprintf("%s cities [flags]", exec),
		ShortHelp:  "List cities (app backend).",
		FlagSet:    citiesFS,
		Exec: func(ctx context.Context, _ []string) error {
			cities, err := client.Cities(ctx)
			if err != nil {
				return err
			}

			if flagCitiesJSON {
				return writeJSON(stdout, cities)
			}

			table := newTable(stdout, "Code", "Name")
			for _, c := range cities {
				table.Append([]string{strconv.Itoa(c.CityCode), c.Name})
			}
			table.Render()
			return nil
		},
	}

	rootCmd := &ffcli.Command{
		ShortUsage:  fmt.Sprintf("%s <subcommand> [flags]", exec),
		ShortHelp:   "Client for EPDK's Şarj@TR EV charging platform.",
		Subcommands: []*ffcli.Command{stationsCmd, stationCmd, registryCmd, citiesCmd},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}

	return rootCmd.ParseAndRun(ctx, osargs[1:])
}

// registryRow is the common subset of a gateway and a mirror station.
type registryRow struct {
	no, brand, name, operator string
	lat, lng                  float64
	sockets                   int
	maxKW                     float64
	raw                       any
}

func gatewayRow(s RegistryStation) registryRow {
	maxkw := 0.0
	for _, sk := range s.Sockets {
		maxkw = max(maxkw, sk.Power)
	}
	return registryRow{
		no: s.No, brand: s.Brand, name: s.Name, operator: s.NetworkOperator,
		lat: s.Lat, lng: s.Lng, sockets: len(s.Sockets), maxKW: maxkw, raw: s,
	}
}

func mirrorRow(s MirrorStation) registryRow {
	maxkw := 0.0
	for _, sk := range s.Sockets {
		if sk.KW != nil {
			maxkw = max(maxkw, *sk.KW)
		}
	}
	return registryRow{
		no: s.No, brand: s.Brand, name: s.Name, operator: s.Operator,
		lat: s.Lat, lng: s.Lng, sockets: len(s.Sockets), maxKW: maxkw, raw: s,
	}
}

func parseAt(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, Istanbul); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("expected YYYY-MM-DD[ HH:MM], got %q", s)
}

func writeJSON(w io.Writer, v any) error {
	return json.MarshalWrite(w, v, jsontext.WithIndent("  "), jsontext.Multiline(true))
}

func newTable(w io.Writer, headers ...string) *tablewriter.Table {
	table := tablewriter.NewWriter(w)
	table.SetHeader(headers)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	return table
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func mapz[T any, U any](items []T, fn func(T) U) []U {
	result := make([]U, 0, len(items))
	for _, item := range items {
		result = append(result, fn(item))
	}
	return result
}

func filter[T any](items []T, fn func(T) bool) []T {
	var result []T
	for _, item := range items {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}
