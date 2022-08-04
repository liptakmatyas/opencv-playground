package main

import (
	"context"
	"image/color"
	"strings"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gocv.io/x/gocv"
)

func guiWithFileSourceMain(parentCtx context.Context, args *CliArgs) error {
	// Create app.

	webcamfx := app.NewWithID("webcamfx")
	webcamfx.SetIcon(theme.ColorPaletteIcon())

	// Create app window.

	window := webcamfx.NewWindow("WebcamFX")
	window.SetFixedSize(true)
	window.SetMaster()

	// Create GUI components.

	toolbar := widget.NewToolbar()

	var fxMakers fxMakerFuncs = nil
	var fxParams []SettingsContainerMaker = nil

	switch strings.ToLower(args.Fx) {
	case "blur":
		blurParams := NewFaceBlurrerParameters(args.ClassifierFile)
		fxParams = []SettingsContainerMaker{blurParams}

		fxMakers = fxMakerFuncs{
			func(inChan <-chan *gocv.Mat) FrameStreamerNode {
				return NewFaceBlurrer("BLUR", inChan, blurParams).TransformerNode
			},
		}

	case "bgrm":
		bgSubParams := NewBackgroundSubtractorParameters()
		thrshParams := NewThresholderParameters(
			25.0,
			"d",
		)
		cntrrParams := NewContourerParameters(
			5000.0,
			color.RGBA{255.0, 255.0, 255.0, 255.0},
			"",
		)
		bgColParams := NewBackgroundColorizerParameters(
			gocv.Scalar{Val1: 0.0, Val2: 255.0, Val3: 0.0, Val4: 255.0},
		)

		fxParams = []SettingsContainerMaker{
			bgSubParams,
			thrshParams,
			cntrrParams,
			bgColParams,
		}

		var (
			cloner *ImageStreamCloner
		)

		fxMakers = fxMakerFuncs{
			func(inChan <-chan *gocv.Mat) FrameStreamerNode {
				cloner = NewImageStreamCloner("CLNR", inChan)
				return cloner
			},
			func(inChan <-chan *gocv.Mat) FrameStreamerNode {
				return NewBackgroundSubtractor("BGSUB", inChan, bgSubParams)
			},
			func(inChan <-chan *gocv.Mat) FrameStreamerNode {
				return NewThresholder("THRSH", inChan, thrshParams).TransformerNode
			},
			func(inChan <-chan *gocv.Mat) FrameStreamerNode {
				cntrr := NewContourer("CNTRR", inChan, cntrrParams).TransformerNode
				return cntrr
			},
			func(inChan <-chan *gocv.Mat) FrameStreamerNode {
				return NewBackgroundColorizer("BGCOL", cloner.Clone(), inChan, bgColParams).CombinerNode
			},
		}
	}

	vsParams := NewVideoSourceParameters(args.SourceId, true, false)
	vsMaker := func(name string, vsp *VideoSourceParameters) FrameStreamerNode {
		return NewVideoFileSource(name, vsp)
	}

	logger.Debugf("Create ISP") // FIXME: Sometimes start-up hangs after this log message...???
	isp := NewImageStreamPlayer(window, toolbar, vsMaker, vsParams, fxMakers, fxParams)
	logger.Debugf("ISP created")

	toolbar.Append(isp.ToolbarAction())
	toolbar.Append(widget.NewToolbarSeparator())
	toolbar.Append(widget.NewToolbarSpacer())

	mainView := container.New(layout.NewHBoxLayout(),
		isp.SettingsContainer(),
		isp.ViewContainer(),
	)

	// Populate window.
	window.SetContent(container.New(layout.NewVBoxLayout(),
		toolbar,
		mainView,
	))

	// Run background loop.
	ctx, cancelCtx := context.WithCancel(parentCtx)

	logger.Tracef("Starting ISP.")
	isp.Run(ctx)
	ispErr := make(chan error, 1)
	go func() {
		ispErr <- <-isp.Err()
		window.Close()
	}()

	// Start GUI.
	logger.Infof("Starting GUI application.")
	window.ShowAndRun()
	cancelCtx()
	logger.Tracef("GUI application stopped.")

	// Shutdown.
	logger.Tracef("Waiting for the ISP to stop...")
	err := <-ispErr
	if err != nil {
		logger.WithError(err).Error("ISP run failed.")
	}
	logger.Tracef("ISP stopped.")

	logger.Infof("Shutdown complete.")
	return nil
}
