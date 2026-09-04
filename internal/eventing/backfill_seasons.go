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
	animeepisode "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime_episode"
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
// TheTVDB cannot be asked directly: its series carry remote ids for TheMovieDB
// and IMDB only, and its season records carry none at all, so nothing on that
// side knows a MyAnimeList entry exists. The only keys the two systems share
// are air dates.
//
// So this joins on those, and only where the join is exact. Our 2006 Haruhi
// row's episodes aired 2006-04-03, 04-10, 04-17 ... and TheTVDB season 1's
// episodes aired on those same days; that is a join, not a guess. Anything
// short of agreement writes nothing and leaves the season unknown, because a
// blank field is honest and a wrong season is not.

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
	// RequiredRatio is the share of an anime's dated episodes that must land on
	// a season's air days for that season to be accepted.
	RequiredRatio float64
}

const (
	backfillPageSize      = 100
	defaultRequiredRatio  = 0.8
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

// seasonAirDays is the set of calendar days each season aired on, which is what
// the episode match joins against.
func seasonAirDays(episodes []thetvdb_api.EpisodeBaseRecord) map[int]map[string]bool {
	days := map[int]map[string]bool{}
	for _, ep := range episodes {
		if ep.SeasonNumber == nil || ep.Aired == nil {
			continue
		}
		aired, err := time.Parse(thetvdbDateLayout, *ep.Aired)
		if err != nil {
			continue
		}
		if days[*ep.SeasonNumber] == nil {
			days[*ep.SeasonNumber] = map[string]bool{}
		}
		days[*ep.SeasonNumber][dayKey(aired)] = true
	}

	return days
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

// dayKey reduces a timestamp to the calendar day, which is the granularity
// both sides agree on: TheTVDB publishes dates, we store timestamps.
func dayKey(t time.Time) string {
	return t.UTC().Format(thetvdbDateLayout)
}

// matchByEpisodes picks the season whose episodes aired on the same days as
// this anime's.
//
// This is the rule that makes the result data rather than a guess. It needs
// most of our episodes accounted for, and it needs one season to explain them
// better than any other -- a series whose seasons were re-cut, or an anime
// whose episode list we only half hold, produces a tie and is left alone.
//
// requiredRatio is deliberately not 1.0: TheTVDB and MyAnimeList disagree about
// a recap or a delayed broadcast often enough that demanding every episode
// would reject correct matches. It is high enough that a season can only win by
// actually being the season.
func matchByEpisodes(ourAired []time.Time, seasonDays map[int]map[string]bool, requiredRatio float64) (int, bool) {
	if len(ourAired) == 0 {
		return 0, false
	}

	ours := map[string]bool{}
	for _, t := range ourAired {
		ours[dayKey(t)] = true
	}

	bestSeason, bestHits, runnerUp := 0, 0, 0
	for season, days := range seasonDays {
		hits := 0
		for day := range ours {
			if days[day] {
				hits++
			}
		}
		if hits > bestHits {
			bestSeason, bestHits, runnerUp = season, hits, bestHits
		} else if hits > runnerUp {
			runnerUp = hits
		}
	}

	if bestHits == 0 {
		return 0, false
	}
	// Ambiguity is a refusal. Two seasons explaining our episodes equally well
	// means we cannot tell which one this is.
	if bestHits == runnerUp {
		return 0, false
	}
	if float64(bestHits)/float64(len(ours)) < requiredRatio {
		return 0, false
	}

	return bestSeason, true
}

// matchByExactStart is the fallback for anime whose episodes we do not hold.
//
// Exact equality, not proximity: the anime's first day must be the day a
// season began, and exactly one season must begin on it. A "closest season
// within N days" rule would happily label a spin-off that premiered near a
// season boundary, which is the failure this whole command exists to avoid.
func matchByExactStart(start time.Time, windows []seasonWindow) (int, bool) {
	season, found := 0, 0
	for _, w := range windows {
		if !w.first.IsZero() && dayKey(w.first) == dayKey(start) {
			season = w.number
			found++
		}
	}

	if found != 1 {
		return 0, false
	}

	return season, true
}

func BackfillSeasons(opts BackfillSeasonsOptions) error {
	if opts.RequiredRatio <= 0 {
		opts.RequiredRatio = defaultRequiredRatio
	}

	cfg := config.LoadConfigOrPanic()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

	api := thetvdb_api.NewTheTVDBApi(cfg.TheTVDBConfig, &http.Client{})
	database := db.NewDB(cfg.DBConfig)
	animeRepo := anime2.NewAnimeRepository(database)
	episodeRepo := animeepisode.NewAnimeEpisodeRepository(database)
	linkRepo := thetvdblink.NewTheTVDBLinkRepository(database)

	// One fetch per series, not per anime. A series with 76 anime pointing at
	// it -- Pokemon does -- would otherwise be fetched 76 times.
	type seriesSeasons struct {
		windows []seasonWindow
		days    map[int]map[string]bool
	}
	seriesCache := map[string]seriesSeasons{}

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
		zap.Float64("requiredRatio", opts.RequiredRatio))

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
			series, cached := seriesCache[seriesID]
			if !cached {
				data, err := api.GetEpisodesBySeriesID(ctx, seriesID)
				if err != nil {
					log.Warn("Failed to fetch series episodes",
						zap.String("seriesID", seriesID), zap.String("error", err.Error()))
					failed++
					continue
				}
				series = seriesSeasons{
					windows: seasonWindows(data.Episodes),
					days:    seasonAirDays(data.Episodes),
				}
				seriesCache[seriesID] = series

				if opts.DelayMs > 0 {
					time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
				}
			}

			// Episodes first: agreement on air days is evidence, a start date
			// landing on a season boundary is only a coincidence that usually
			// holds.
			ourEpisodes, err := episodeRepo.FindByAnimeID(ctx, record.ID)
			if err != nil {
				log.Warn("Failed to read episodes",
					zap.String("animeID", record.ID), zap.String("error", err.Error()))
				failed++
				continue
			}
			aired := make([]time.Time, 0, len(ourEpisodes))
			for _, e := range ourEpisodes {
				if e.Aired != nil {
					aired = append(aired, *e.Aired)
				}
			}

			seasonNumber, matched := matchByEpisodes(aired, series.days, opts.RequiredRatio)
			how := "episodes"
			if !matched {
				seasonNumber, matched = matchByExactStart(start, series.windows)
				how = "exact-start"
			}
			if !matched {
				skipped++
				continue
			}

			name := fmt.Sprintf("season %d", seasonNumber)
			if record.TitleEn != nil && *record.TitleEn != "" {
				name = fmt.Sprintf("%s (season %d)", *record.TitleEn, seasonNumber)
			}

			log.Info("Matched season",
				zap.String("animeID", record.ID),
				zap.String("seriesID", seriesID),
				zap.Int("season", seasonNumber),
				zap.String("how", how),
				zap.Int("ourDatedEpisodes", len(aired)),
				zap.String("start", start.Format(thetvdbDateLayout)))

			if opts.DryRun {
				linked++
				continue
			}

			if err := linkRepo.Upsert(ctx, &thetvdblink.TheTVDBLink{
				AnimeID:      record.ID,
				TheTVDBID:    seriesID,
				SeasonNumber: seasonNumber,
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
