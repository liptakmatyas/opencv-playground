package main

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type ImageStreamViewer struct {
	*SinkNode[image.Image]
	p *ImageStreamViewerParameters
}

var _ Node = &ImageStreamViewer{}

func NewImageStreamViewer(name string, inChan <-chan image.Image, container *fyne.Container, viewObjIdx int, p *ImageStreamViewerParameters) *ImageStreamViewer {
	isv := &ImageStreamViewer{
		SinkNode: NewSinkNode[image.Image](name, inChan),
		p:        p,
	}

	var viewImg *canvas.Image

	isv.SetupFunc(func() error {
		return nil
	})

	isv.StepFunc(func(img image.Image) error {
		if !isv.p.enabled || img == nil {
			return nil
		}

		if viewImg != nil {
			viewImg.Image = img
		} else {
			imgBounds := img.Bounds()
			imgSize := fyne.NewSize(float32(imgBounds.Dx()), float32(imgBounds.Dy()))
			viewImg = canvas.NewImageFromImage(img)
			viewImg.SetMinSize(imgSize)
			viewImg.FillMode = canvas.ImageFillOriginal
			container.Objects[viewObjIdx] = viewImg
		}

		container.Refresh()

		return nil
	})

	return isv
}

type ImageStreamViewerParameters struct {
	enabled bool
}

var _ SettingsContainerMaker = &BackgroundColorizerParameters{}

func NewImageStreamViewerParameters(enabled bool) *ImageStreamViewerParameters {
	return &ImageStreamViewerParameters{
		enabled: enabled,
	}
}

func (p *ImageStreamViewerParameters) MakeSettingsContainer(win fyne.Window) *fyne.Container {
	previewEnabledCheck := widget.NewCheckWithData("Preview", binding.BindBool(&p.enabled))

	return container.New(layout.NewCenterLayout(),
		previewEnabledCheck,
	)
}
