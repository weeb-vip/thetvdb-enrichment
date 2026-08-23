package commands

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/weeb-vip/thetvdb-enrichment/internal/eventing"
)

// serveNatsCmd is the NATS counterpart of serve-kafka.
//
// A separate command rather than a flag: prod and staging run the same image,
// so the command name is what selects the transport, keeping that choice in the
// deployment values rather than an environment variable.
var serveNatsCmd = &cobra.Command{
	Use:   "serve-nats",
	Short: "Consume TheTVDB enrichment events from NATS JetStream",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("Running TheTVDB enrichment eventing over NATS...")

		return eventing.EventingNats()
	},
}

func init() {
	rootCmd.AddCommand(serveNatsCmd)
}
