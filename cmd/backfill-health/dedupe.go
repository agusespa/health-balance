package main

import (
	"strings"

	"health-balance/internal/ingest"
)

// dedupeSources cleans the raw export before it is aggregated. The export
// holds every device's own samples, so the Watch and the iPhone both count the
// same walk; a Shortcut that groups by day gets totals Health has already
// de-duplicated. Per day this keeps:
//   - summed keys (steps, exercise, energy, nutrition, sets): only the source
//     with the highest total
//   - sleep: only the Watch's segments when it recorded asleep stages,
//     otherwise only the source with the most asleep time
//
// Other keys (averages and latest readings) are left as they are.
func dedupeSources(samples []ingest.Sample) []ingest.Sample {
	type group struct{ key, day string }
	totals := map[group]map[string]float64{}
	add := func(g group, src string, v float64) {
		if totals[g] == nil {
			totals[g] = map[string]float64{}
		}
		totals[g][src] += v
	}
	groupOf := func(s ingest.Sample) (group, bool) {
		switch {
		case s.Key == ingest.KeySleep:
			return group{s.Key, ingest.NightOf(s.Start)}, true
		case ingest.Summed(s.Key):
			return group{s.Key, s.Start.Format("2006-01-02")}, true
		}
		return group{}, false
	}

	for _, s := range samples {
		g, ok := groupOf(s)
		switch {
		case !ok:
		case s.Key == ingest.KeySleep:
			if ingest.Asleep(s.Stage) {
				add(g, s.Source, s.End.Sub(s.Start).Hours())
			}
		default:
			add(g, s.Source, s.Value)
		}
	}

	keep := map[group]string{}
	for g, bySource := range totals {
		best := ""
		for src, total := range bySource {
			switch {
			case best == "":
				best = src
			case g.key == ingest.KeySleep && isWatch(src) != isWatch(best):
				if isWatch(src) {
					best = src
				}
			case total > bySource[best] || (total == bySource[best] && src < best):
				best = src
			}
		}
		keep[g] = best
	}

	out := samples[:0]
	for _, s := range samples {
		if g, ok := groupOf(s); ok {
			if best, has := keep[g]; has && s.Source != best {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

func isWatch(source string) bool {
	return strings.Contains(strings.ToLower(source), "watch")
}
