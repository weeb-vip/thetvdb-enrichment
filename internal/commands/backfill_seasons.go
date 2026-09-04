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
	Long: `Walks anime carrying a thetvdbid, fetches the series' episodes once per
series, and matches each anime's start date to the season whose first episode is
closest.

An anime whose nearest season is further away than the tolerance gets no link,
leaving season_number null. That is deliberate: an unknown season renders as
nothing, a wrong one renders as a lie.

Season 0 is TheTVDB's specials season and is matched like any other; the UI
renders it as Special.

Flags:
  --dry-run          match and log, write nothing
  --limit N          stop after N anime (0 = all)
  --delay-ms N       pause after each series fetch, to stay under rate limits
  --after ID         resume from an anime id (exclusive)
  --tolerance-days N how far a start date may sit from a season's first
                     episode and still count as a match (default 45)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		limit, _ := cmd.Flags().GetInt("limit")
		delayMs, _ := cmd.Flags().GetInt("delay-ms")
		after, _ := cmd.Flags().GetString("after")
		tolerance, _ := cmd.Flags().GetInt("tolerance-days")

		return eventing.BackfillSeasons(eventing.BackfillSeasonsOptions{
			DryRun:        dryRun,
			Limit:         limit,
			DelayMs:       delayMs,
			After:         after,
			ToleranceDays: tolerance,
		})
	},
}

func init() {
	backfillSeasons.Flags().Bool("dry-run", false, "match and log, write nothing")
	backfillSeasons.Flags().Int("limit", 0, "stop after N anime (0 = all)")
	backfillSeasons.Flags().Int("delay-ms", 200, "pause after each series fetch")
	backfillSeasons.Flags().String("after", "", "resume from an anime id (exclusive)")
	backfillSeasons.Flags().Int("tolerance-days", 45, "max gap between start date and a season's first episode")
	rootCmd.AddCommand(backfillSeasons)
}
