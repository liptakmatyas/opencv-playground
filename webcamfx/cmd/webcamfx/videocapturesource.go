package main

import (
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

const (
	flipBoth      int = -1
	flipUpDown    int = 0
	flipLeftRight int = 1
)

type VideoCaptureSource struct {
	*SourceNode[*gocv.Mat]
	p *VideoSourceParameters
}

var (
	_ Node              = &VideoCaptureSource{}
	_ FrameStreamerNode = &VideoCaptureSource{}
)

func NewVideoCaptureSource(name string, p *VideoSourceParameters) *VideoCaptureSource {
	vcs := &VideoCaptureSource{
		SourceNode: NewSourceNode[*gocv.Mat](name),
		p:          p,
	}

	var (
		videoCapture *gocv.VideoCapture
		frameBuffer  gocv.Mat
	)

	vcs.SetupFunc(func() error {
		var err error

		// NOTE: This turns on the video capture.
		//  If the source is a (web) camera, then it means that the camera starts recording. The status
		//  indicator LED of the camera should also turn on.
		videoCapture, err = gocv.OpenVideoCapture(vcs.p.sourceId)
		if err != nil {
			logger.WithError(err).Errorf("OpenVideoCapture failed")
			return errors.Wrapf(err, "failed to open video capture source '%s'", vcs.p.sourceId)
		}

		frameBuffer = gocv.NewMat()

		return nil
	})

	vcs.TeardownFunc(func() error {
		return flattenErrors(
			errors.Wrapf(videoCapture.Close(), "video capture source teardown error"),
			errors.Wrapf(frameBuffer.Close(), "video capture frame buffer teardown error"),
		)
	})

	vcs.StepFunc(func() (*gocv.Mat, error) {
		if ok := videoCapture.Read(&frameBuffer); !ok {
			return nil, io.EOF
		}

		if vcs.p.lrFlip && vcs.p.udFlip {
			gocv.Flip(frameBuffer, &frameBuffer, flipBoth)
		} else if vcs.p.lrFlip {
			gocv.Flip(frameBuffer, &frameBuffer, flipLeftRight)
		} else if vcs.p.udFlip {
			gocv.Flip(frameBuffer, &frameBuffer, flipUpDown)
		}

		// NOTE: Returned frame may be empty!
		return &frameBuffer, nil
	})

	return vcs
}

type VideoSourceParameters struct {
	sourceId string
	lrFlip   bool
	udFlip   bool
}

var _ SettingsContainerMaker = &VideoSourceParameters{}

func NewVideoSourceParameters(sourceId string, lrFlip bool, udFlip bool) *VideoSourceParameters {
	return &VideoSourceParameters{
		sourceId: sourceId,
		lrFlip:   lrFlip,
		udFlip:   udFlip,
	}
}

func (p *VideoSourceParameters) MakeSettingsContainer(_ fyne.Window) *fyne.Container {
	lrFlipCheck := widget.NewCheckWithData("LR", binding.BindBool(&p.lrFlip))
	udFlipCheck := widget.NewCheckWithData("UD", binding.BindBool(&p.udFlip))

	return container.New(layout.NewGridLayout(2),
		widget.NewLabel("SourceId:"),
		widget.NewLabel(p.sourceId),

		widget.NewLabel("Flip:"),
		container.New(layout.NewHBoxLayout(),
			lrFlipCheck,
			udFlipCheck,
		),
	)
}
