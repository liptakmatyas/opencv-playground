package main

import (
	"context"
	"image"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func guiMain(parentCtx context.Context, args *CliArgs) error {
	ctx, cancelCtx := context.WithCancel(parentCtx)

	logger := logger.
		WithField("func", "guiMain")

	webcamfx := app.New()
	webcamfx.SetIcon(theme.ColorPaletteIcon())

	window := webcamfx.NewWindow("WebcamFX")
	window.SetFixedSize(true)

	toolbar := widget.NewToolbar()

	rfch := make(chan image.Image)
	is := NewImageStream("is", args.SourceId, rfch)
	isv := NewImageStreamViewer("isv", toolbar, is, rfch)

	toolbar.Append(widget.NewToolbarSeparator())
	toolbar.Append(widget.NewToolbarSpacer())

	statusBar := container.New(layout.NewHBoxLayout(),
		widget.NewLabel("SourceId:"),
		widget.NewLabel(args.SourceId),
		widget.NewSeparator(),
	)

	window.SetContent(container.New(layout.NewVBoxLayout(), toolbar, isv.Container(), statusBar))

	logger.Tracef("Starting ISV SM.")
	isvSmRunErrChan := make(chan error, 1)
	go func() {
		isvSmRunErrChan <- isv.sm.Run(ctx)
	}()

	logger.Infof("Starting GUI application.")
	window.ShowAndRun()
	logger.Tracef("GUI application stopped.")

	logger.Tracef("Waiting for the ISV SM to stop...")
	cancelCtx()
	err := <-isvSmRunErrChan
	if err != nil {
		logger.WithError(err).Error("ISV SM run failed.")
	}
	logger.Tracef("ISV SM stopped.")

	logger.Infof("Shutdown complete.")
	return nil
}
