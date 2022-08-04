package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type BackgroundColorizer struct {
	*CombinerNode[*gocv.Mat, *gocv.Mat, *gocv.Mat]
	p *BackgroundColorizerParameters
}

var _ Node = &BackgroundColorizer{}

func NewBackgroundColorizer(name string, imgChan <-chan *gocv.Mat, maskChan <-chan *gocv.Mat, p *BackgroundColorizerParameters) *BackgroundColorizer {
	bgcol := &BackgroundColorizer{
		CombinerNode: NewCombinerNode[*gocv.Mat, *gocv.Mat, *gocv.Mat](name, imgChan, maskChan),
		p:            p,
	}

	var (
		frameBuffer gocv.Mat
	)

	bgcol.SetupFunc(func() error {
		frameBuffer = gocv.NewMat()
		return nil
	})

	bgcol.TeardownFunc(func() error {
		return errors.Wrapf(frameBuffer.Close(), "frame buffer teardown error")
	})

	bgcol.StepFunc(func(img *gocv.Mat, mask *gocv.Mat) (*gocv.Mat, error) {
		if img.Empty() {
			return nil, nil
		}

		if err := frameBuffer.Close(); err != nil {
			return nil, errors.Wrapf(err, "failed to close previous frame buffer")
		}

		frameBuffer = gocv.NewMatWithSizeFromScalar(bgcol.p.backgroundColor, img.Rows(), img.Cols(), img.Type())
		gocv.BitwiseAndWithMask(*img, *img, &frameBuffer, *mask)
		return &frameBuffer, nil
	})

	return bgcol
}

type BackgroundColorizerParameters struct {
	backgroundColor gocv.Scalar
}

var _ SettingsContainerMaker = &BackgroundColorizerParameters{}

func NewBackgroundColorizerParameters(
	backgroundColor gocv.Scalar,
) *BackgroundColorizerParameters {
	return &BackgroundColorizerParameters{
		backgroundColor: backgroundColor,
	}
}

func (p *BackgroundColorizerParameters) MakeSettingsContainer(win fyne.Window) *fyne.Container {
	bgColorRect := &canvas.Rectangle{
		FillColor:   colorFromScalar(p.backgroundColor),
		StrokeColor: color.NRGBA{0xff, 0xff, 0xff, 0xff},
		StrokeWidth: 1,
	}

	bgColorButton := widget.NewButtonWithIcon("", theme.ColorPaletteIcon(), func() {
		picker := dialog.NewColorPicker("Pick a Color", "Pick background color", func(c color.Color) {
			p.backgroundColor = scalarFromColor(c)
			bgColorRect.FillColor = c
			bgColorRect.Refresh()
		}, win)
		picker.Advanced = true
		picker.Show()
	})

	return container.NewGridWithColumns(2,
		widget.NewLabel("Background color:"),
		// NOTE: This "hack" allows to change the background color of the button.
		container.New(layout.NewMaxLayout(), bgColorRect, bgColorButton),
	)
}
