package main

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type Thresholder struct {
	*TransformerNode[*gocv.Mat]
	p *ThresholderParameters
}

var _ Node = &Thresholder{}

func NewThresholder(name string, inChan <-chan *gocv.Mat, p *ThresholderParameters) *Thresholder {
	thrsh := &Thresholder{
		TransformerNode: NewTransformerNode[*gocv.Mat](name, inChan),
		p:               p,
	}

	var (
		imgThresh gocv.Mat
		kernel    gocv.Mat
	)

	thrsh.SetupFunc(func() error {
		imgThresh = gocv.NewMat()
		kernel = gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
		return nil
	})

	thrsh.TeardownFunc(func() error {
		return flattenErrors(
			errors.Wrapf(kernel.Close(), "image kernel teardowm error"),
			errors.Wrapf(imgThresh.Close(), "imgThresh buffer teardown error"),
		)
	})

	thrsh.StepFunc(func(img *gocv.Mat) (*gocv.Mat, error) {
		if img.Empty() {
			return nil, nil
		}

		gocv.Threshold(*img, &imgThresh, float32(thrsh.p.threshold), 255, gocv.ThresholdBinary)
		if imgThresh.Empty() {
			return nil, nil
		}
		runOps(thrsh.p.opsOnRawThreshold, &imgThresh, &kernel)

		return &imgThresh, nil
	})

	return thrsh
}

type ThresholderParameters struct {
	threshold         float64
	opsOnRawThreshold string
}

var _ SettingsContainerMaker = &ThresholderParameters{}

func NewThresholderParameters(
	threshold float64,
	opsOnRawThreshold string,
) *ThresholderParameters {
	return &ThresholderParameters{
		threshold:         threshold,
		opsOnRawThreshold: opsOnRawThreshold,
	}
}

func (p *ThresholderParameters) MakeSettingsContainer(_ fyne.Window) *fyne.Container {
	thresholdData := binding.BindFloat(&p.threshold)
	thresholdLabel := widget.NewLabelWithData(binding.FloatToStringWithFormat(thresholdData, "%03.0f"))
	thresholdEntry := widget.NewSliderWithData(0.0, 255.0, thresholdData)

	opsOnRawThresholdData := binding.BindString(&p.opsOnRawThreshold)
	opsOnRawThresholdEntry := widget.NewEntryWithData(opsOnRawThresholdData)
	opsOnRawThresholdEntry.Validator = validation.NewRegexp(`^[ed]*$`, "Can only contain characters: ed")

	return container.NewGridWithColumns(2,
		container.NewHBox(widget.NewLabel("Threshold:"), thresholdLabel),
		thresholdEntry,

		widget.NewLabel("Threshold ops:"),
		opsOnRawThresholdEntry,
	)
}
