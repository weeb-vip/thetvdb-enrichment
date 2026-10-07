package commands

import (
	"github.com/spf13/cobra"
	"github.com/weeb-vip/thetvdb-enrichment/internal/eventing"
)

// syncPosters publishes the tall 2:3 series poster for every anime carrying a
// thetvdbid, alongside the wide background artwork sync-banners publishes.
//
// A sibling command rather than a flag on sync-banners: the two are one code
// path internally, but "sync-banners --artwork poster" reads as a contradiction
// in a k8s manifest and in the log line that records what a run did. The name
// should say what the run produced.
//
// These land at /posters/<id>, deliberately not at the bucket root. The root
// holds the scraper's MyAnimeList image, which MAL serves at 225px wide -- fine
// in a list row, and roughly a 7.8x upscale filling a phone hero. Keeping them
// apart means a show TheTVDB does not carry keeps the image it already has.
var syncPosters = &cobra.Command{
	Use:   "sync-posters",
	Short: "Publish series poster artwork for every anime that has a thetvdbid",
	Long: `Walks anime rows carrying a thetvdbid, resolves the series poster
(TheTVDB artwork type 2) and publishes it to the image-sync topic, where it is
stored under /posters/<id>.

An anime with no poster of the right shape publishes nothing, so the frontend
falls back to the image it already has rather than being handed a wide banner
where it asked for a tall one.

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
		opts, err := artworkOptions(cmd, eventing.ArtworkPoster)
		if err != nil {
			return err
		}
		return eventing.SyncBanners(opts)
	},
}

func init() {
	rootCmd.AddCommand(syncPosters)
	artworkFlags(syncPosters)
}
