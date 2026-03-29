package cmd

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/arastu/irtrafficgen/internal/version"
)

func Run(ctx context.Context, args []string) error {
	return NewRootCommand().Run(ctx, args)
}

func runFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "config",
			Value: "",
			Usage: "YAML config file path",
		},
		&cli.BoolFlag{
			Name:  "live",
			Usage: "perform real HTTPS/DNS (overrides config dry_run)",
		},
		&cli.StringFlag{
			Name:  "dry-run",
			Usage: "true|false overrides config dry_run; omit to use config",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "print session metrics summary when run stops",
		},
		&cli.BoolFlag{
			Name:  "once",
			Usage: "single request then exit",
		},
	}
}

func NewRootCommand() *cli.Command {
	return &cli.Command{
		Name:            "irtrafficgen",
		Usage:           "Iran-focused traffic generator for authorized lab testing",
		Description:     "Uses embedded geosite.dat and geoip.dat. See README and research.md.",
		Version:         version.Version,
		HideVersion:     true,
		DefaultCommand:  "run",
		Suggest:         true,
		Flags:           runFlags(),
		Commands:        []*cli.Command{newInspectCommand(), newRunCommand(), newVersionCommand()},
		Reader:          os.Stdin,
		Writer:          os.Stdout,
		ErrWriter:       os.Stderr,
	}
}
