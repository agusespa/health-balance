package ingest

import (
	"strings"
	"testing"
	"time"
)

const exportSample = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE HealthData [
<!ELEMENT HealthData (ExportDate,Me,(Record|Correlation|Workout)*)>
<!ATTLIST HealthData locale CDATA #REQUIRED>
]>
<HealthData locale="en_SE">
 <ExportDate value="2026-11-25 19:58:01 +0100"/>
 <Me HKCharacteristicTypeIdentifierBiologicalSex="HKBiologicalSexMale"/>
 <Record type="HKQuantityTypeIdentifierRestingHeartRate" sourceName="Agus’ Apple Watch" unit="count/min" creationDate="2026-11-19 00:04:00 +0100" startDate="2026-11-18 00:03:14 +0100" endDate="2026-11-18 23:54:13 +0100" value="53"/>
 <Record type="HKQuantityTypeIdentifierHeartRateVariabilitySDNN" sourceName="Agustin’s Apple Watch" unit="ms" startDate="2026-11-18 03:00:00 +0100" endDate="2026-11-18 03:01:00 +0100" value="41.5">
  <HeartRateVariabilityMetadataList>
   <InstantaneousBeatsPerMinute bpm="55" time="3:00:01,12 AM"/>
  </HeartRateVariabilityMetadataList>
 </Record>
 <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Mi Fit" unit="count/min" startDate="2020-01-01 10:00:00 +0100" endDate="2020-01-01 10:00:00 +0100" value="70"/>
 <Record type="HKCategoryTypeIdentifierSleepAnalysis" sourceName="Agustin’s Apple Watch" device="&lt;&lt;HKDevice: 0x1&gt;, name:Apple Watch&gt;" startDate="2026-11-17 23:30:00 +0100" endDate="2026-11-18 06:30:00 +0100" value="HKCategoryValueSleepAnalysisAsleepCore"/>
 <Record type="HKCategoryTypeIdentifierMindfulSession" sourceName="Insight Timer" startDate="2026-11-18 07:00:00 +0100" endDate="2026-11-18 07:10:00 +0100" value="HKCategoryValueNotApplicable"/>
 <Record type="HKQuantityTypeIdentifierBodyMass" sourceName="Health" unit="stone" startDate="2026-11-18 07:00:00 +0100" endDate="2026-11-18 07:00:00 +0100" value="12"/>
 <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30"/>
</HealthData>`

func TestReadExport(t *testing.T) {
	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	var samples []Sample
	stats, err := ReadExport(strings.NewReader(exportSample), stockholm, func(s Sample) { samples = append(samples, s) })
	if err != nil {
		t.Fatal(err)
	}
	if got := stats.ExportDate.Format("2006-01-02"); got != "2026-11-25" {
		t.Errorf("export date %s", got)
	}
	if stats.Records != 6 {
		t.Errorf("records %d, want 6", stats.Records)
	}
	if len(samples) != 4 {
		t.Fatalf("used %d samples, want 4 (heart rate ignored, stone rejected): %+v", len(samples), samples)
	}
	if len(stats.Rejected) != 1 {
		t.Errorf("rejected %v, want the stone weight", stats.Rejected)
	}

	daily := map[string]float64{}
	for _, dv := range Aggregate(samples, Options{}) {
		daily[dv.Date+" "+dv.Metric] = dv.Value
	}
	want := map[string]float64{
		"2026-11-18 rhr":              53,
		"2026-11-18 hrv_sdnn":         41.5,
		"2026-11-18 sleep_hours":      7,
		"2026-11-18 mindful_sessions": 1,
		"2026-11-18 mindful_min":      10,
	}
	for k, v := range want {
		if daily[k] != v {
			t.Errorf("%s = %v, want %v", k, daily[k], v)
		}
	}
}

func TestReadExportReportsBrokenXML(t *testing.T) {
	_, err := ReadExport(strings.NewReader(`<HealthData><Record type="x"`), time.UTC, func(Sample) {})
	if err == nil {
		t.Error("want an error for truncated XML")
	}
}

// The export writes every time as +0100. Records tagged with HKTimeZone set
// the zone for everything after them until the next tag.
const exportZones = `<HealthData>
 <ExportDate value="2026-11-25 19:58:01 +0100"/>
 <Record type="HKQuantityTypeIdentifierStepCount" sourceName="Agus' Apple Watch" unit="count" startDate="2021-11-02 02:30:00 +0100" endDate="2021-11-02 02:40:00 +0100" value="100"/>
 <Record type="HKCategoryTypeIdentifierSleepAnalysis" sourceName="Agus' Apple Watch" startDate="2021-11-02 03:00:00 +0100" endDate="2021-11-02 10:00:00 +0100" value="HKCategoryValueSleepAnalysisAsleepUnspecified">
  <MetadataEntry key="HKTimeZone" value="America/Argentina/Buenos_Aires"/>
 </Record>
 <Workout workoutActivityType="HKWorkoutActivityTypeWalking" startDate="2025-06-01 08:00:00 +0100" endDate="2025-06-01 09:00:00 +0100">
  <MetadataEntry key="HKTimeZone" value="Europe/Stockholm"/>
 </Workout>
 <Record type="HKCategoryTypeIdentifierSleepAnalysis" sourceName="Agustin’s Apple Watch" startDate="2025-06-01 22:30:00 +0100" endDate="2025-06-02 05:30:00 +0100" value="HKCategoryValueSleepAnalysisAsleepCore"/>
</HealthData>`

func TestReadExportConvertsToRecordedZone(t *testing.T) {
	var samples []Sample
	if _, err := ReadExport(strings.NewReader(exportZones), time.UTC, func(s Sample) { samples = append(samples, s) }); err != nil {
		t.Fatal(err)
	}
	if len(samples) != 3 {
		t.Fatalf("got %d samples", len(samples))
	}
	want := []string{
		"2021-11-01 22:30 -03",  // before the first tag: uses the first tag's zone
		"2021-11-01 23:00 -03",  // Argentina, 4 h behind the export's +0100
		"2025-06-01 23:30 CEST", // Stockholm summer time, 1 h ahead of +0100
	}
	for i, s := range samples {
		if got := s.Start.Format("2006-01-02 15:04 MST"); got != want[i] {
			t.Errorf("sample %d starts %s, want %s", i, got, want[i])
		}
	}

	daily := map[string]float64{}
	for _, dv := range Aggregate(samples, Options{}) {
		daily[dv.Date+" "+dv.Metric] = dv.Value
	}
	if daily["2021-11-01 steps"] != 100 {
		t.Errorf("steps should land on 2021-11-01 local: %v", daily)
	}
	if daily["2021-11-02 sleep_bedtime"] != 300 { // 23:00 local
		t.Errorf("bedtime %v, want 300 (23:00)", daily["2021-11-02 sleep_bedtime"])
	}
	if daily["2025-06-02 sleep_bedtime"] != 330 { // 23:30 local
		t.Errorf("bedtime %v, want 330 (23:30)", daily["2025-06-02 sleep_bedtime"])
	}
}
