package eventing

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/weeb-vip/thetvdb-enrichment/config"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/thetvdblink"
	"github.com/weeb-vip/thetvdb-enrichment/internal/logger"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
)

// BackfillSeasons works out which season of its series each anime is.
//
// MyAnimeList files each broadcast run as its own anime; TheTVDB keeps them as
// seasons of one series, so our thetvdbid is identical across every run of a
// show. 1,756 TheTVDB ids are shared by 6,425 of our anime -- 76 of them are
// Pokemon -- and nothing until now said which run any of them was.
//
// The air dates say it. Haruhi's TheTVDB season 1 ran 2006-04-03 to 2006-07-03
// and season 2 ran 2009-05-22 to 2009-09-11; our two rows carry exactly those
// dates. Matching a start date to a season's window is enough, and where it is
// not obviously enough this refuses to guess.

// seasonWindow is one season's air range, as TheTVDB reports it.
type seasonWindow struct {
	number int
	first  time.Time
	last   time.Time
	count  int
}

type BackfillSeasonsOptions struct {
	DryRun  bool
	Limit   int
	DelayMs int
	After   string
	// ToleranceDays is how far an anime's start date may sit from a season's
	// first episode and still be called a match.
	ToleranceDays int
}

const (
	backfillPageSize      = 100
	defaultToleranceDays  = 45
	thetvdbDateLayout     = "2006-01-02"
	animeStartDateLayouts = "2006-01-02T15:04:05Z07:00|2006-01-02 15:04:05-07|2006-01-02"
)

// parseFlexibleDate copes with the three shapes start_date arrives in.
//
// The column is text and has been written by more than one scraper version, so
// it holds RFC3339, postgres' own timestamptz rendering, and bare dates.
func parseFlexibleDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	for _, layout := range strings.Split(animeStartDateLayouts, "|") {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}

	// Last resort: the leading date of anything date-shaped.
	if len(value) >= 10 {
		if t, err := time.Parse(thetvdbDateLayout, value[:10]); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

// seasonWindows groups a series' episodes into per-season air ranges.
func seasonWindows(episodes []thetvdb_api.EpisodeBaseRecord) []seasonWindow {
	type acc struct {
		first, last time.Time
		count       int
	}
	byNumber := map[int]*acc{}

	for _, ep := range episodes {
		if ep.SeasonNumber == nil {
			continue
		}
		entry, ok := byNumber[*ep.SeasonNumber]
		if !ok {
			entry = &acc{}
			byNumber[*ep.SeasonNumber] = entry
		}
		entry.count++

		if ep.Aired == nil {
			continue
		}
		aired, err := time.Parse(thetvdbDateLayout, *ep.Aired)
		if err != nil {
			continue
		}
		if entry.first.IsZero() || aired.Before(entry.first) {
			entry.first = aired
		}
		if entry.last.IsZero() || aired.After(entry.last) {
			entry.last = aired
		}
	}

	windows := make([]seasonWindow, 0, len(byNumber))
	for number, entry := range byNumber {
		windows = append(windows, seasonWindow{
			number: number,
			first:  entry.first,
			last:   entry.last,
			count:  entry.count,
		})
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].number < windows[j].number })

	return windows
}

// matchSeason picks the season whose first episode sits closest to the anime's
// start date, or reports no match.
//
// Closest-start rather than "start falls inside the window", because a run that
// begins a few days before its first listed episode is common and an interval
// test would reject it. The tolerance is what stops that becoming a guess: a
// show whose nearest season is months away gets no link at all, which is the
// honest answer and leaves the field null rather than wrong.
func matchSeason(start time.Time, windows []seasonWindow, toleranceDays int) (seasonWindow, bool) {
	best := seasonWindow{}
	found := false
	var bestGap time.Duration

	for _, window := range windows {
		if window.first.IsZero() {
			continue
		}
		gap := start.Sub(window.first)
		if gap < 0 {
			gap = -gap
		}
		if !found || gap < bestGap {
			best, bestGap, found = window, gap, true
		}
	}

	if !found || bestGap > time.Duration(toleranceDays)*24*time.Hour {
		return seasonWindow{}, false
	}

	return best, true
}

func BackfillSeasons(opts BackfillSeasonsOptions) error {
	if opts.ToleranceDays <= 0 {
		opts.ToleranceDays = defaultToleranceDays
	}

	cfg := config.LoadConfigOrPanic()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

	api := thetvdb_api.NewTheTVDBApi(cfg.TheTVDBConfig, &http.Client{})
	database := db.NewDB(cfg.DBConfig)
	animeRepo := anime2.NewAnimeRepository(database)
	linkRepo := thetvdblink.NewTheTVDBLinkRepository(database)

	// One fetch per series, not per anime. A series with 76 anime pointing at
	// it -- Pokemon does -- would otherwise be fetched 76 times.
	seriesCache := map[string][]seasonWindow{}

	var (
		after     = opts.After
		processed int
		linked    int
		skipped   int
		failed    int
	)

	log.Info("Starting season backfill",
		zap.Bool("dryRun", opts.DryRun),
		zap.Int("limit", opts.Limit),
		zap.Int("toleranceDays", opts.ToleranceDays))

	for {
		records, err := animeRepo.FindWithTheTVDBID(ctx, after, backfillPageSize)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}

		for _, record := range records {
			after = record.ID
			if opts.Limit > 0 && processed >= opts.Limit {
				break
			}
			processed++

			if record.TheTVDBID == nil || *record.TheTVDBID == "" || record.StartDate == nil {
				skipped++
				continue
			}

			start, ok := parseFlexibleDate(*record.StartDate)
			if !ok {
				skipped++
				continue
			}

			seriesID := *record.TheTVDBID
			windows, cached := seriesCache[seriesID]
			if !cached {
				data, err := api.GetEpisodesBySeriesID(ctx, seriesID)
				if err != nil {
					log.Warn("Failed to fetch series episodes",
						zap.String("seriesID", seriesID), zap.String("error", err.Error()))
					failed++
					continue
				}
				windows = seasonWindows(data.Episodes)
				seriesCache[seriesID] = windows

				if opts.DelayMs > 0 {
					time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
				}
			}

			window, matched := matchSeason(start, windows, opts.ToleranceDays)
			if !matched {
				skipped++
				continue
			}

			name := fmt.Sprintf("season %d", window.number)
			if record.TitleEn != nil && *record.TitleEn != "" {
				name = fmt.Sprintf("%s (season %d)", *record.TitleEn, window.number)
			}

			log.Info("Matched season",
				zap.String("animeID", record.ID),
				zap.String("seriesID", seriesID),
				zap.Int("season", window.number),
				zap.String("start", start.Format(thetvdbDateLayout)),
				zap.String("seasonFirstAired", window.first.Format(thetvdbDateLayout)))

			if opts.DryRun {
				linked++
				continue
			}

			if err := linkRepo.Upsert(ctx, &thetvdblink.TheTVDBLink{
				AnimeID:      record.ID,
				TheTVDBID:    seriesID,
				SeasonNumber: window.number,
				Name:         &name,
			}); err != nil {
				log.Error("Failed to write link",
					zap.String("animeID", record.ID), zap.String("error", err.Error()))
				failed++
				continue
			}
			linked++
		}

		if opts.Limit > 0 && processed >= opts.Limit {
			break
		}
	}

	log.Info("Season backfill finished",
		zap.Int("processed", processed),
		zap.Int("linked", linked),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
		zap.Int("seriesFetched", len(seriesCache)))

	return nil
}
