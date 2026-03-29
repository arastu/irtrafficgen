package cmd

import (
	"context"

	"github.com/arastu/irtrafficgen/internal/applog"
	"github.com/arastu/irtrafficgen/internal/version"
	"github.com/urfave/cli/v3"
)

func newVersionCommand() *cli.Command {
	return &cli.Command{
		Name:        "version",
		Usage:       "Print release version and commit",
		Description: "Values are set at link time via -ldflags (see Makefile and release workflow).",
		Action:      versionAction,
	}
}

func versionAction(_ context.Context, _ *cli.Command) error {
	log, err := applog.New()
	if err != nil {
		return cli.Exit(err, 2)
	}
	defer func() { _ = log.Sync() }()
	applog.Version(log, version.Version, version.Commit)
	return nil
}
