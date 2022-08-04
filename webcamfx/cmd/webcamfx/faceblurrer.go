package main

import (
	"image"
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type FaceBlurrer struct {
	*TransformerNode[*gocv.Mat]
	p *FaceBlurrerParameters
}

var _ Node = &FaceBlurrer{}

func NewFaceBlurrer(name string, inChan <-chan *gocv.Mat, p *FaceBlurrerParameters) *FaceBlurrer {
	fb := &FaceBlurrer{
		TransformerNode: NewTransformerNode[*gocv.Mat](name, inChan),
		p:               p,
	}

	var classifier gocv.CascadeClassifier

	fb.SetupFunc(func() error {
		classifier = gocv.NewCascadeClassifier()
		if ok := classifier.Load(p.classifierFile); !ok {
			return errors.Errorf("Failed to load classifier file '%s'", p.classifierFile)
		}
		return nil
	})

	fb.TeardownFunc(func() error {
		return classifier.Close()
	})

	fb.StepFunc(func(img *gocv.Mat) (*gocv.Mat, error) {
		rects := classifier.DetectMultiScale(*img)
		for _, r := range rects {
			faceRegion := img.Region(r)
			gocv.GaussianBlur(faceRegion, &faceRegion, image.Pt(75, 75), 0, 0, gocv.BorderDefault)
			err := faceRegion.Close()
			if err != nil {
				return nil, errors.Wrap(err, "failed to close face region")
			}
		}

		return img, nil
	})

	return fb
}

type FaceBlurrerParameters struct {
	classifierFile string
}

var _ SettingsContainerMaker = &FaceBlurrerParameters{}

func NewFaceBlurrerParameters(classifierFile string) *FaceBlurrerParameters {
	return &FaceBlurrerParameters{
		classifierFile: classifierFile,
	}
}

func (p *FaceBlurrerParameters) MakeSettingsContainer(_ fyne.Window) *fyne.Container {
	return container.New(layout.NewVBoxLayout(),
		container.New(layout.NewHBoxLayout(),
			widget.NewLabel("Classifier file:"),
			widget.NewLabel(path.Base(p.classifierFile)),
		),
	)
}
