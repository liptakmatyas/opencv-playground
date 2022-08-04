package main

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type Contourer struct {
	*TransformerNode[*gocv.Mat]
	p *ContourerParameters
}

var _ Node = &Contourer{}

func NewContourer(name string, inChan <-chan *gocv.Mat, p *ContourerParameters) *Contourer {
	cntrr := &Contourer{
		TransformerNode: NewTransformerNode[*gocv.Mat](name, inChan),
		p:               p,
	}

	var (
		frameBuffer gocv.Mat
		kernel      gocv.Mat
	)

	cntrr.SetupFunc(func() error {
		frameBuffer = gocv.NewMat()
		kernel = gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
		return nil
	})

	cntrr.TeardownFunc(func() error {
		return flattenErrors(
			errors.Wrapf(kernel.Close(), "image kernel teardowm error"),
			errors.Wrapf(frameBuffer.Close(), "frame buffer teardown error"),
		)
	})

	cntrr.StepFunc(func(img *gocv.Mat) (*gocv.Mat, error) {
		if img.Empty() {
			return nil, nil
		}

		img.CopyTo(&frameBuffer)
		contours := gocv.FindContours(*img, gocv.RetrievalExternal, gocv.ChainApproxSimple)
		for i := 0; i < contours.Size(); i++ {
			area := gocv.ContourArea(contours.At(i))
			if area < cntrr.p.minArea {
				continue
			}

			gocv.DrawContours(&frameBuffer, contours, i, cntrr.p.contourColor, -1)
		}
		contours.Close()

		runOps(cntrr.p.postContourOps, &frameBuffer, &kernel)

		return &frameBuffer, nil
	})

	return cntrr
}

type ContourerParameters struct {
	minArea             float64
	contourColor        color.RGBA
	postContourOps      string
	validPostContourOps string
}

var _ SettingsContainerMaker = &ContourerParameters{}

func NewContourerParameters(
	minArea float64,
	contourColor color.RGBA,
	opsOnContours string,
) *ContourerParameters {
	return &ContourerParameters{
		minArea:             minArea,
		contourColor:        contourColor,
		postContourOps:      opsOnContours,
		validPostContourOps: AllImageOps,
	}
}

func (p *ContourerParameters) MakeSettingsContainer(_ fyne.Window) *fyne.Container {
	minAreaData := binding.BindFloat(&p.minArea)
	minAreaEntry := widget.NewEntryWithData(binding.FloatToStringWithFormat(minAreaData, "%.0f"))
	minAreaEntry.Validator = validation.NewRegexp(`^\d+$`, "Must be a positive integer")

	isPostContourOpValid := validation.NewRegexp(p.validPostContourOps, UnknownImageOpErrMsg(p.validPostContourOps))

	postContourOpsUserData := binding.BindString(&p.postContourOps)
	postContourOpsUserEntry := widget.NewEntryWithData(postContourOpsUserData)
	postContourOpsUserEntry.Validator = isPostContourOpValid

	return container.NewGridWithColumns(2,
		widget.NewLabel("Minimum area:"),
		minAreaEntry,

		widget.NewLabel("PostContourOps:"),
		postContourOpsUserEntry,
	)
}
