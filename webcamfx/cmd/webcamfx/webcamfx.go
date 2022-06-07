package main

import (
	"fmt"
	_ "image/png"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// CliArgs stores the parsed command line arguments.
type CliArgs struct {
	// SourceId identifies the source for the frames for GoCV.
	// It can be a device ID, a file name, a URL, etc.
	// See https://pkg.go.dev/gocv.io/x/gocv#OpenVideoCapture
	SourceId string

	// LogLevelString can be used to override the default log level.
	LogLevelString string

	// logLevel is the numeric representation of the log level.
	logLevel logrus.Level
}

func (args *CliArgs) Validate() error {
	err := args.ValidateLogLevelString()
	if err != nil {
		return err
	}

	return nil
}

func (args *CliArgs) ValidateLogLevelString() error {
	l, err := logrus.ParseLevel(args.LogLevelString)
	if err != nil {
		return err
	}

	args.logLevel = l
	return nil
}

func main() {
	args := &CliArgs{
		SourceId:       "0",
		LogLevelString: "INFO",
		logLevel:       logrus.InfoLevel,
	}

	app := &cli.App{
		Before: func(c *cli.Context) error {
			err := args.ValidateLogLevelString()
			if err != nil {
				return err
			}

			initLogger(args.logLevel)
			logger.Infof("Running with arguments: %+v", *args)
			return nil
		},

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       fmt.Sprintf("log level: [%s]", allLogLevels),
				Value:       args.LogLevelString,
				Destination: &args.LogLevelString,
			},
		},

		Commands: []*cli.Command{
			{
				Name:  "gui",
				Usage: "Run GUI application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "source",
						Aliases:     []string{"s"},
						Usage:       "source frame stream; e.g., device ID, file name, URL, etc.",
						Value:       args.SourceId,
						Destination: &args.SourceId,
					},
				},
				Before: func(c *cli.Context) error {
					return args.Validate()
				},
				Action: func(c *cli.Context) error {
					return guiMain(c.Context, args)
				},
			},

			{
				Name:  "gfunc",
				Usage: "Run gfunc example",
				Action: func(c *cli.Context) error {
					return gfuncMain(c.Context)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.
			WithError(err).
			Fatalf("Application failed.")
	}
}
