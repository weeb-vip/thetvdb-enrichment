package eventing

import (
	"testing"
	"time"

	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
)

func ep(season int, aired string) thetvdb_api.EpisodeBaseRecord {
	s := season
	a := aired

	return thetvdb_api.EpisodeBaseRecord{SeasonNumber: &s, Aired: &a}
}

// The real shape of TheTVDB series 79414, which is the case this was written
// for: MyAnimeList splits it into two anime, TheTVDB keeps it as one series
// with two seasons, and only the air dates tell them apart.
func haruhiWindows() []seasonWindow {
	return seasonWindows([]thetvdb_api.EpisodeBaseRecord{
		ep(0, "2010-02-06"),
		ep(1, "2006-04-03"), ep(1, "2006-05-15"), ep(1, "2006-07-03"),
		ep(2, "2009-05-22"), ep(2, "2009-06-19"), ep(2, "2009-09-11"),
	})
}

func TestSeasonWindowsGroupsBySeason(t *testing.T) {
	windows := haruhiWindows()

	if len(windows) != 3 {
		t.Fatalf("want 3 seasons, got %d", len(windows))
	}
	if windows[0].number != 0 || windows[1].number != 1 || windows[2].number != 2 {
		t.Fatalf("seasons out of order: %+v", windows)
	}
	if got := windows[1].first.Format(thetvdbDateLayout); got != "2006-04-03" {
		t.Errorf("season 1 first aired = %s, want 2006-04-03", got)
	}
	if got := windows[1].last.Format(thetvdbDateLayout); got != "2006-07-03" {
		t.Errorf("season 1 last aired = %s, want 2006-07-03", got)
	}
}

func TestMatchSeasonPicksTheRightRun(t *testing.T) {
	windows := haruhiWindows()

	for _, tc := range []struct {
		name  string
		start string
		want  int
	}{
		{"the 2006 anime is season 1", "2006-04-03", 1},
		{"the 2009 anime is season 2", "2009-05-22", 2},
		{"the 2010 film is the specials season", "2010-02-06", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, _ := time.Parse(thetvdbDateLayout, tc.start)
			got, ok := matchSeason(start, windows, defaultToleranceDays)
			if !ok {
				t.Fatalf("no match for %s", tc.start)
			}
			if got.number != tc.want {
				t.Errorf("season = %d, want %d", got.number, tc.want)
			}
		})
	}
}

// The guard that keeps this from inventing seasons. An anime sharing a series
// id but airing years from any known season -- a spin-off, a recap special the
// series record does not list -- must come back unmatched so the column stays
// null rather than wrong.
func TestMatchSeasonRefusesWhenNothingIsClose(t *testing.T) {
	start, _ := time.Parse(thetvdbDateLayout, "2015-10-01")

	if _, ok := matchSeason(start, haruhiWindows(), defaultToleranceDays); ok {
		t.Error("matched a season five years from any air date; want no match")
	}
}

func TestMatchSeasonIgnoresSeasonsWithNoAirDates(t *testing.T) {
	s := 3
	windows := seasonWindows([]thetvdb_api.EpisodeBaseRecord{
		ep(1, "2006-04-03"),
		{SeasonNumber: &s}, // announced, nothing aired
	})
	start, _ := time.Parse(thetvdbDateLayout, "2006-04-05")

	got, ok := matchSeason(start, windows, defaultToleranceDays)
	if !ok || got.number != 1 {
		t.Errorf("got season %d ok=%v, want season 1", got.number, ok)
	}
}

func TestParseFlexibleDateHandlesEveryShapeStartDateArrivesIn(t *testing.T) {
	for _, in := range []string{
		"2006-04-03",
		"2006-04-03T04:00:00Z",
		"2006-04-03 04:00:00+00",
	} {
		got, ok := parseFlexibleDate(in)
		if !ok {
			t.Errorf("%q did not parse", in)
			continue
		}
		if got.Format(thetvdbDateLayout) != "2006-04-03" {
			t.Errorf("%q parsed to %s", in, got.Format(thetvdbDateLayout))
		}
	}

	if _, ok := parseFlexibleDate(""); ok {
		t.Error("empty string parsed")
	}
}
