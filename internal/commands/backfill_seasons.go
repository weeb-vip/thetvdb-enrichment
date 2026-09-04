package commands

import (
	"github.com/spf13/cobra"
	"github.com/weeb-vip/thetvdb-enrichment/internal/eventing"
)

// backfillSeasons fills in which season of its series each anime is.
//
// MyAnimeList files every broadcast run as a separate anime while TheTVDB keeps
// them as seasons of one series, so our thetvdbid is identical across runs:
// 1,756 ids are shared by 6,425 anime. The season is recoverable from the air
// dates, which is what this does.
//
// Writes thetvdb_link rather than anime.season_number directly. That table is
// the curated source -- the admin panel writes it by hand -- and a trigger
// mirrors it onto the anime row, so there is one writer to the column rather
// than two disagreeing.
var backfillSeasons = &cobra.Command{
	Use:   "backfill-seasons",
	Short: "Derive each anime's series season from TheTVDB air dates",
	Long: `Walks anime carrying a thetvdbid, fetches the series' episodes once per series,
and matches on the days our episodes aired against the days each season aired.

TheTVDB cannot be asked directly -- its series carry remote ids for TheMovieDB
and IMDB only, its seasons carry none, so nothing there knows a MyAnimeList
entry exists. Air dates are the only key the two systems share.

A season is accepted only when it explains most of an anime's episodes and
explains them better than any other season. Anything ambiguous, or short of the
ratio, writes nothing and leaves season_number null: an unknown season renders
as nothing, a wrong one renders as a lie.

Anime whose episodes we do not hold fall back to exact equality between the
anime's start date and a season's first air date -- exact, not nearest.

Season 0 is TheTVDB's specials season and is matched like any other; the UI
renders it as Special.

Flags:
  --dry-run          match and log, write nothing
  --limit N          stop after N anime (0 = all)
  --delay-ms N       pause after each series fetch, to stay under rate limits
  --after ID         resume from an anime id (exclusive)
  --required-ratio F share of an anime's dated episodes that must land on a
                     season's air days for it to be accepted (default 0.8)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		limit, _ := cmd.Flags().GetInt("limit")
		delayMs, _ := cmd.Flags().GetInt("delay-ms")
		after, _ := cmd.Flags().GetString("after")
		ratio, _ := cmd.Flags().GetFloat64("required-ratio")

		return eventing.BackfillSeasons(eventing.BackfillSeasonsOptions{
			DryRun:        dryRun,
			Limit:         limit,
			DelayMs:       delayMs,
			After:         after,
			RequiredRatio: ratio,
		})
	},
}

func init() {
	backfillSeasons.Flags().Bool("dry-run", false, "match and log, write nothing")
	backfillSeasons.Flags().Int("limit", 0, "stop after N anime (0 = all)")
	backfillSeasons.Flags().Int("delay-ms", 200, "pause after each series fetch")
	backfillSeasons.Flags().String("after", "", "resume from an anime id (exclusive)")
	backfillSeasons.Flags().Float64("required-ratio", 0.8, "share of an anime's episodes that must land on a season's air days")
	rootCmd.AddCommand(backfillSeasons)
}
