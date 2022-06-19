package main

import (
	"fmt"
	_ "image/png"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// CliArgs stores the parsed command line arguments.
type CliArgs struct {
	// SourceId identifies the source for the frames for GoCV.
	// It can be a device ID, a file name, a URL, etc.
	// See https://pkg.go.dev/gocv.io/x/gocv#OpenVideoCapture
	SourceId string

	// ClassifierFile points to the classifier file to be loaded for the face detection.
	ClassifierFile string

	// Fx identifies the effect to apply to the source.
	Fx string

	// LogLevelString can be used to override the default log level.
	LogLevelString string

	// logLevel is the numeric representation of the log level.
	logLevel logrus.Level
}

func (args *CliArgs) Validate() error {
	err := args.ValidateFx()
	if err != nil {
		return err
	}

	err = args.ValidateLogLevelString()
	if err != nil {
		return err
	}

	return nil
}

func (args *CliArgs) ValidateFx() error {
	switch args.Fx {
	case "none", "blur", "bgrm":
		return nil
	default:
		return errors.Errorf("unknown fx '%s'", args.Fx)
	}
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
		ClassifierFile: "",
		Fx:             "none",
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

				Subcommands: []*cli.Command{
					{
						Name:  "none",
						Usage: "Apply no effects",

						Before: func(c *cli.Context) error {
							args.Fx = "none"
							return args.Validate()
						},

						Action: func(c *cli.Context) error {
							logger.Infof("Running with arguments: %+v", *args)
							return guiMain(c.Context, args)
						},
					},

					{
						Name:  "blur",
						Usage: "Apply blur effect",

						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "cls-file",
								Aliases:     []string{"c"},
								Usage:       "classifier file to use; e.g., ./haarcascade_frontalface_default.xml",
								Required:    true,
								Destination: &args.ClassifierFile,
							},
						},

						Before: func(c *cli.Context) error {
							args.Fx = "blur"
							return args.Validate()
						},

						Action: func(c *cli.Context) error {
							logger.Infof("Running with arguments: %+v", *args)
							return guiMain(c.Context, args)
						},
					},

					{
						Name:  "bgrm",
						Usage: "Apply background removal effect",

						Before: func(c *cli.Context) error {
							args.Fx = "bgrm"
							return args.Validate()
						},

						Action: func(c *cli.Context) error {
							logger.Infof("Running with arguments: %+v", *args)
							return guiMain(c.Context, args)
						},
					},
				},
			},

			{
				Name:  "guiFromFile",
				Usage: "Run GUI application from file",

				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "source",
						Aliases:     []string{"s"},
						Usage:       "source frame stream; e.g., device ID, file name, URL, etc.",
						Value:       args.SourceId,
						Destination: &args.SourceId,
					},
				},

				Subcommands: []*cli.Command{
					{
						Name:  "none",
						Usage: "Apply no effects",

						Before: func(c *cli.Context) error {
							args.Fx = "none"
							return args.Validate()
						},

						Action: func(c *cli.Context) error {
							logger.Infof("Running with arguments: %+v", *args)
							return guiWithFileSourceMain(c.Context, args)
						},
					},

					{
						Name:  "blur",
						Usage: "Apply blur effect",

						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "cls-file",
								Aliases:     []string{"c"},
								Usage:       "classifier file to use; e.g., ./haarcascade_frontalface_default.xml",
								Required:    true,
								Destination: &args.ClassifierFile,
							},
						},

						Before: func(c *cli.Context) error {
							args.Fx = "blur"
							return args.Validate()
						},

						Action: func(c *cli.Context) error {
							logger.Infof("Running with arguments: %+v", *args)
							return guiWithFileSourceMain(c.Context, args)
						},
					},

					{
						Name:  "bgrm",
						Usage: "Apply background removal effect",

						Before: func(c *cli.Context) error {
							args.Fx = "bgrm"
							return args.Validate()
						},

						Action: func(c *cli.Context) error {
							logger.Infof("Running with arguments: %+v", *args)
							return guiWithFileSourceMain(c.Context, args)
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Application failed:", err.Error())
	}
}

// TODO: Move to some "errorutils" package... And make it better. :)
func flattenErrors(errs ...error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}

	var finalErr error
	for _, err := range errs {
		if err == nil {
			continue
		}

		if finalErr != nil {
			finalErr = errors.Errorf("%v, %v", finalErr, err)
		} else {
			finalErr = err
		}
	}
	return finalErr
}
