package ingest

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// exportTypes maps Apple Health export record types to Shortcut keys.
// Everything else in the export is ignored.
var exportTypes = map[string]string{
	"HKQuantityTypeIdentifierVO2Max":                     KeyVO2,
	"HKQuantityTypeIdentifierRestingHeartRate":           KeyRHR,
	"HKQuantityTypeIdentifierHeartRateVariabilitySDNN":   KeyHRV,
	"HKQuantityTypeIdentifierStepCount":                  KeySteps,
	"HKQuantityTypeIdentifierAppleExerciseTime":          KeyExercise,
	"HKQuantityTypeIdentifierBodyMass":                   KeyWeight,
	"HKQuantityTypeIdentifierWaistCircumference":         KeyWaist,
	"HKQuantityTypeIdentifierBloodPressureSystolic":      KeyBPSys,
	"HKQuantityTypeIdentifierBloodPressureDiastolic":     KeyBPDia,
	"HKQuantityTypeIdentifierHeartRateRecoveryOneMinute": KeyCardio,
	"HKCategoryTypeIdentifierSleepAnalysis":              KeySleep,
	"HKCategoryTypeIdentifierMindfulSession":             KeyMindful,
}

// ExportStats describes one pass over an export.
type ExportStats struct {
	ExportDate time.Time
	Records    int            // all <Record> elements
	Used       map[string]int // samples handed to fn, per key
	Rejected   map[string]int // mapped records that failed to convert, per reason
	Zones      map[string]int // samples per time zone they were converted to
}

type zonePoint struct {
	at   time.Time
	zone string
}

// ReadExport streams an Apple Health export.xml and calls fn for each record
// of a type health-balance uses, in file order.
//
// The export writes every timestamp in one fixed offset (the exporting phone's
// at export time), so summer times are an hour off and data recorded abroad
// lands on the wrong hours. Many records carry an HKTimeZone metadata entry;
// ReadExport builds a timeline from those and converts each sample to the zone
// in effect when it was recorded. Samples before the first tagged record use
// that first zone; with no tags at all, fallback is used.
func ReadExport(r io.Reader, fallback *time.Location, fn func(Sample)) (ExportStats, error) {
	stats := ExportStats{Used: map[string]int{}, Rejected: map[string]int{}, Zones: map[string]int{}}
	var samples []Sample
	var zones []zonePoint
	var current time.Time // start of the Record or Workout being read

	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return stats, fmt.Errorf("export.xml offset %d: %w", dec.InputOffset(), err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch el.Name.Local {
		case "ExportDate":
			if t, err := parseTime(attr(el, "value")); err == nil {
				stats.ExportDate = t
			}
		case "Workout":
			current, _ = parseTime(attr(el, "startDate"))
		case "MetadataEntry":
			if attr(el, "key") == "HKTimeZone" && !current.IsZero() {
				zones = append(zones, zonePoint{current, attr(el, "value")})
			}
		case "Record":
			stats.Records++
			current, _ = parseTime(attr(el, "startDate"))
			key, ok := exportTypes[attr(el, "type")]
			if !ok {
				continue
			}
			s, err := NewSample(key, attr(el, "value"), attr(el, "unit"), attr(el, "startDate"), attr(el, "endDate"), attr(el, "sourceName"))
			if err != nil {
				stats.Rejected[key+": "+err.Error()]++
				continue
			}
			samples = append(samples, s)
		}
	}

	locate := zoneTimeline(zones, fallback)
	for _, s := range samples {
		loc := locate(s.Start)
		s.Start, s.End = s.Start.In(loc), s.End.In(loc)
		stats.Used[s.Key]++
		stats.Zones[loc.String()]++
		fn(s)
	}
	return stats, nil
}

// zoneTimeline returns a lookup of the zone in effect at a time: the zone of
// the latest tagged record at or before it.
func zoneTimeline(points []zonePoint, fallback *time.Location) func(time.Time) *time.Location {
	locs := map[string]*time.Location{}
	var valid []zonePoint
	for _, p := range points {
		if _, ok := locs[p.zone]; !ok {
			loc, err := time.LoadLocation(p.zone)
			if err != nil {
				continue
			}
			locs[p.zone] = loc
		}
		valid = append(valid, p)
	}
	sort.SliceStable(valid, func(i, j int) bool { return valid[i].at.Before(valid[j].at) })

	return func(t time.Time) *time.Location {
		if len(valid) == 0 {
			return fallback
		}
		i := sort.Search(len(valid), func(i int) bool { return valid[i].at.After(t) })
		if i == 0 {
			return locs[valid[0].zone]
		}
		return locs[valid[i-1].zone]
	}
}

func attr(el xml.StartElement, name string) string {
	for _, a := range el.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
