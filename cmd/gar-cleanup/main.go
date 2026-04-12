package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/senet/gar-cleanup/internal/engine"
	"github.com/senet/gar-cleanup/internal/policy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		policyFile string
		live       bool
		project    string
		location   string
	)

	root := &cobra.Command{
		Use:   "gar-cleanup",
		Short: "Google Artifact Registry image lifecycle manager",
		Long: `gar-cleanup removes stale container images from Google Artifact Registry
according to a YAML policy. Dry-run is the default; pass --live to delete.`,
	}

	run := &cobra.Command{
		Use:   "run",
		Short: "Execute a cleanup run (dry-run by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			pol, err := policy.LoadFile(policyFile)
			if err != nil {
				return fmt.Errorf("loading policy: %w", err)
			}

			cfg := engine.Config{
				Project:    project,
				Location:   location,
				DryRun:     !live,
				PolicyFile: policyFile,
			}

			eng, err := engine.New(cfg, pol)
			if err != nil {
				return fmt.Errorf("creating engine: %w", err)
			}

			return eng.Run(context.Background())
		},
	}

	run.Flags().StringVar(&policyFile, "policy", "policy.yaml", "Path to YAML policy file")
	run.Flags().BoolVar(&live, "live", false, "Execute live deletions (default is dry-run)")
	run.Flags().StringVar(&project, "project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "GCP project ID")
	run.Flags().StringVar(&location, "location", "us-central1", "GAR location")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gar-cleanup %s (%s) built %s\n", version, commit, date)
		},
	}

	root.AddCommand(run, versionCmd)
	return root
}
