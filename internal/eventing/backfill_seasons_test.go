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

func days(t *testing.T, values ...string) []time.Time {
	t.Helper()
	out := make([]time.Time, 0, len(values))
	for _, v := range values {
		parsed, err := time.Parse(thetvdbDateLayout, v)
		if err != nil {
			t.Fatalf("bad test date %q: %v", v, err)
		}
		out = append(out, parsed)
	}

	return out
}

// TheTVDB series 79414 as the API actually returns it. This is the case the
// command exists for: MyAnimeList splits it into two anime, TheTVDB keeps one
// series with two seasons, and only the air dates connect them.
func haruhi() []thetvdb_api.EpisodeBaseRecord {
	return []thetvdb_api.EpisodeBaseRecord{
		ep(0, "2010-02-06"),
		ep(1, "2006-04-03"), ep(1, "2006-04-10"), ep(1, "2006-04-17"), ep(1, "2006-07-03"),
		ep(2, "2009-05-22"), ep(2, "2009-06-19"), ep(2, "2009-06-26"), ep(2, "2009-07-03"),
		ep(2, "2009-07-10"), ep(2, "2009-07-17"), ep(2, "2009-07-24"), ep(2, "2009-07-31"),
		ep(2, "2009-08-07"), ep(2, "2009-09-11"),
	}
}

func TestMatchByEpisodesJoinsOnAirDays(t *testing.T) {
	airDays := seasonAirDays(haruhi())

	for _, tc := range []struct {
		name string
		ours []time.Time
		want int
	}{
		{
			name: "the 2006 anime is season 1",
			ours: days(t, "2006-04-03", "2006-04-10", "2006-04-17", "2006-07-03"),
			want: 1,
		},
		{
			name: "the 2009 anime is season 2",
			ours: days(t, "2009-05-22", "2009-06-19", "2009-06-26", "2009-07-03",
				"2009-07-10", "2009-07-17", "2009-07-24", "2009-07-31", "2009-08-07", "2009-09-11"),
			want: 2,
		},
		{
			// TheTVDB and MyAnimeList disagree about a delayed broadcast often
			// enough that demanding every episode would reject correct matches.
			name: "nine of ten still matches",
			ours: days(t, "2009-05-22", "2009-06-19", "2009-06-26", "2009-07-03",
				"2009-07-10", "2009-07-17", "2009-07-24", "2009-07-31", "2009-08-07", "2011-01-01"),
			want: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := matchByEpisodes(tc.ours, airDays, defaultRequiredRatio)
			if !ok {
				t.Fatalf("no match")
			}
			if got != tc.want {
				t.Errorf("season = %d, want %d", got, tc.want)
			}
		})
	}
}

// The refusals. Each of these would previously have been answered with a
// nearest-season guess, and each would have written a wrong season.
func TestMatchByEpisodesRefusesRatherThanGuess(t *testing.T) {
	airDays := seasonAirDays(haruhi())

	for _, tc := range []struct {
		name string
		ours []time.Time
	}{
		{
			name: "a spin-off sharing the series id but airing on its own days",
			ours: days(t, "2015-10-07", "2015-10-14", "2015-10-21"),
		},
		{
			name: "too few of our episodes land on the season's days",
			ours: days(t, "2009-05-22", "2009-06-19", "2011-01-08", "2011-01-15", "2011-01-22"),
		},
		{
			name: "an anime we hold no dated episodes for",
			ours: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if season, ok := matchByEpisodes(tc.ours, airDays, defaultRequiredRatio); ok {
				t.Errorf("matched season %d; want a refusal", season)
			}
		})
	}
}

// A tie is a refusal: if two seasons explain our episodes equally well we
// cannot say which one this anime is.
func TestMatchByEpisodesRefusesATie(t *testing.T) {
	airDays := seasonAirDays([]thetvdb_api.EpisodeBaseRecord{
		ep(1, "2020-01-05"), ep(2, "2020-01-05"),
	})

	if season, ok := matchByEpisodes(days(t, "2020-01-05"), airDays, 1.0); ok {
		t.Errorf("matched season %d on a tie; want a refusal", season)
	}
}

func TestMatchByExactStartRequiresExactness(t *testing.T) {
	windows := seasonWindows(haruhi())

	start := days(t, "2009-05-22")[0]
	if got, ok := matchByExactStart(start, windows); !ok || got != 2 {
		t.Errorf("exact start: got season %d ok=%v, want season 2", got, ok)
	}

	// One day off is not a match. The old rule allowed 45 days of slack, which
	// is how a spin-off premiering near a season boundary got mislabelled.
	off := days(t, "2009-05-23")[0]
	if season, ok := matchByExactStart(off, windows); ok {
		t.Errorf("matched season %d one day off; want a refusal", season)
	}
}

func TestParseFlexibleDateHandlesEveryShapeStartDateArrivesIn(t *testing.T) {
	for _, in := range []string{"2006-04-03", "2006-04-03T04:00:00Z", "2006-04-03 04:00:00+00"} {
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
