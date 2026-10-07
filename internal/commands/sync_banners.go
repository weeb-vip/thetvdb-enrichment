package commands

import (
	"github.com/spf13/cobra"
	"github.com/weeb-vip/thetvdb-enrichment/internal/eventing"
)

// syncBanners backfills artwork for every anime that already carries a
// thetvdbid, without going through the link table.
//
// The normal enrichment path is driven by thetvdb_link rows and, for each one,
// deletes and re-imports that anime's episodes for a specific season. That is
// the right behaviour when a human links a show in the admin panel, but it is
// far too heavy -- and season-sensitive -- to run across a bulk id backfill:
// a wrong season would wipe a show's real episodes. Artwork needs none of that.
// It only needs the series id, so it gets its own path.
var syncBanners = &cobra.Command{
	Use:   "sync-banners",
	Short: "Publish banner artwork for every anime that has a thetvdbid",
	Long: `Walks anime rows carrying a thetvdbid, resolves the series background
artwork from TheTVDB, and publishes it to the image-sync topic.

Does not touch episodes, seasons or the link table.

Flags:
  --dry-run        resolve and log, publish nothing
  --limit N        stop after N anime (0 = all)
  --delay-ms N     pause between anime, to stay under TheTVDB rate limits
  --after ID       resume from an anime id (exclusive)
  --season S       only one season's anime, e.g. FALL_2026
  --ids / --ids-file
                   only the given anime
  --force          ask image-sync to re-pull even artwork it already holds
  --transport T    nats (default) or kafka, wherever image-sync listens`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := artworkOptions(cmd, eventing.ArtworkBanner)
		if err != nil {
			return err
		}
		return eventing.SyncBanners(opts)
	},
}

func init() {
	rootCmd.AddCommand(syncBanners)
	artworkFlags(syncBanners)
}
