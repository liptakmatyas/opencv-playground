package main

import (
	"io"

	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type VideoFileSource struct {
	*SourceNode[*gocv.Mat]
	p *VideoSourceParameters
}

var (
	_ Node              = &VideoFileSource{}
	_ FrameStreamerNode = &VideoFileSource{}
)

func NewVideoFileSource(name string, p *VideoSourceParameters) *VideoFileSource {
	vcs := &VideoFileSource{
		SourceNode: NewSourceNode[*gocv.Mat](name),
		p:          p,
	}

	var (
		src         *gocv.VideoCapture
		frameBuffer gocv.Mat
	)

	vcs.SetupFunc(func() error {
		var err error

		src, err = gocv.VideoCaptureFile(vcs.p.sourceId)
		if err != nil {
			logger.WithError(err).Errorf("OpenVideoCapture failed")
			return errors.Wrapf(err, "failed to open video capture source '%s'", vcs.p.sourceId)
		}

		frameBuffer = gocv.NewMat()

		return nil
	})

	vcs.TeardownFunc(func() error {
		return flattenErrors(
			errors.Wrapf(src.Close(), "video capture source teardown error"),
			errors.Wrapf(frameBuffer.Close(), "video capture frame buffer teardown error"),
		)
	})

	vcs.StepFunc(func() (*gocv.Mat, error) {
		if ok := src.Read(&frameBuffer); !ok {
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
