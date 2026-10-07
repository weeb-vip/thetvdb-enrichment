package commands

import (
	"bufio"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weeb-vip/thetvdb-enrichment/internal/eventing"
)

// artworkFlags are the flags sync-banners and sync-posters share.
func artworkFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("dry-run", false, "resolve artwork but publish nothing")
	cmd.Flags().Int("limit", 0, "stop after N anime (0 = all)")
	cmd.Flags().Int("delay-ms", 250, "pause between anime, in milliseconds")
	cmd.Flags().String("after", "", "resume from this anime id (exclusive)")
	cmd.Flags().String("season", "", "only this season's anime, e.g. FALL_2026")
	cmd.Flags().String("ids", "", "only these anime, comma-separated")
	cmd.Flags().String("ids-file", "", "only the anime listed in this file, one id per line")
	cmd.Flags().Bool("force", false, "ask image-sync to re-pull the artwork even when it already holds it")
	cmd.Flags().String("transport", "nats", "where image-sync listens: nats (default) or kafka")
}

// artworkOptions reads the shared flags into the sync options.
func artworkOptions(cmd *cobra.Command, kind eventing.ArtworkKind) (eventing.SyncBannersOptions, error) {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	limit, _ := cmd.Flags().GetInt("limit")
	delayMs, _ := cmd.Flags().GetInt("delay-ms")
	after, _ := cmd.Flags().GetString("after")
	season, _ := cmd.Flags().GetString("season")
	idsFlag, _ := cmd.Flags().GetString("ids")
	idsFile, _ := cmd.Flags().GetString("ids-file")
	force, _ := cmd.Flags().GetBool("force")
	transport, _ := cmd.Flags().GetString("transport")
	ids, err := collectIDs(idsFlag, idsFile)
	if err != nil {
		return eventing.SyncBannersOptions{}, err
	}
	return eventing.SyncBannersOptions{
		Kind: kind, DryRun: dryRun, Limit: limit, DelayMs: delayMs, After: after,
		Season: season, IDs: ids, Force: force, Transport: transport,
	}, nil
}

// collectIDs merges --ids (comma-separated) and --ids-file (one per line,
// blank lines and # comments ignored), in order, without duplicates.
func collectIDs(idsFlag, idsFile string) ([]string, error) {
	seen := map[string]bool{}
	var ids []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") || seen[s] {
			return
		}
		seen[s] = true
		ids = append(ids, s)
	}
	for _, s := range strings.Split(idsFlag, ",") {
		add(s)
	}
	if idsFile != "" {
		f, err := os.Open(idsFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			add(sc.Text())
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	return ids, nil
}
