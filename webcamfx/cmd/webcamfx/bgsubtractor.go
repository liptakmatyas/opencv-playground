package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type BackgroundSubtractor struct {
	// TODO: *ConverterNode[*gocv.Mat, *FrameWithRegion]
	*ConverterNode[*gocv.Mat, *gocv.Mat]
	p *BackgroundSubtractorParameters
}

var _ Node = &BackgroundSubtractor{}

func NewBackgroundSubtractor(name string, inChan <-chan *gocv.Mat, p *BackgroundSubtractorParameters) *BackgroundSubtractor {
	bgsub := &BackgroundSubtractor{
		ConverterNode: NewConverterNode[*gocv.Mat, *gocv.Mat](name, inChan),
		p:             p,
	}

	var (
		imgDelta gocv.Mat
		mog2     gocv.BackgroundSubtractorMOG2
	)

	bgsub.SetupFunc(func() error {
		imgDelta = gocv.NewMat()
		mog2 = gocv.NewBackgroundSubtractorMOG2()
		return nil
	})

	bgsub.TeardownFunc(func() error {
		return flattenErrors(
			errors.Wrapf(mog2.Close(), "backround subtractor teardown error"),
			errors.Wrapf(imgDelta.Close(), "imgDelta buffer teardown error"),
		)
	})

	bgsub.StepFunc(func(img *gocv.Mat) (*gocv.Mat, error) {
		if img.Empty() {
			return nil, nil
		}

		mog2.Apply(*img, &imgDelta)

		return &imgDelta, nil
	})

	return bgsub
}

type BackgroundSubtractorParameters struct {
	name string
}

var _ SettingsContainerMaker = &BackgroundSubtractorParameters{}

func NewBackgroundSubtractorParameters() *BackgroundSubtractorParameters {
	return &BackgroundSubtractorParameters{
		name: "MOG2",
	}
}

func (p *BackgroundSubtractorParameters) MakeSettingsContainer(_ fyne.Window) *fyne.Container {
	nameData := binding.BindString(&p.name)
	nameEntry := widget.NewEntryWithData(nameData)
	nameEntry.Disable()

	return container.NewGridWithColumns(2,
		widget.NewLabel("Name:"),
		nameEntry,
	)
}
