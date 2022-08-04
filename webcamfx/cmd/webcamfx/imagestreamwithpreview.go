package main

import (
	"image"

	"fyne.io/fyne/v2"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type ImageStreamWithPreview struct {
	*PreviewerNode[*gocv.Mat, image.Image]
}

var _ Node = &ImageStreamWithPreview{}

func NewImageStreamWithPreview(name string, inChan <-chan *gocv.Mat) *ImageStreamWithPreview {
	iswp := &ImageStreamWithPreview{
		PreviewerNode: NewPreviewerNode[*gocv.Mat, image.Image](name, inChan),
	}

	var (
		previewBuffer gocv.Mat
	)

	iswp.SetupFunc(func() error {
		previewBuffer = gocv.NewMat()
		return nil
	})

	iswp.TeardownFunc(func() error {
		return errors.Wrapf(previewBuffer.Close(), "preview buffer teardown error")
	})

	iswp.StepFunc(func(rawFrame *gocv.Mat) (image.Image, error) {
		if rawFrame == nil {
			return nil, nil
		}

		gocv.Resize(*rawFrame, &previewBuffer, image.Pt(160, 120), 0, 0, gocv.InterpolationNearestNeighbor)

		previewImg, err := previewBuffer.ToImage()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert raw frame")
		}

		return previewImg, err
	})

	return iswp
}

func WrapFrameStreamerWithPreview(
	streamer FrameStreamerNode,
	previewContainer *fyne.Container,
	previewObjIdx int,
	previewParams *ImageStreamViewerParameters,
) (*ImageStreamWithPreview, *ImageStreamViewer, FrameStreamerNode) {
	nodeName := streamer.Name() + ":PREVIEW"
	wrapped := NewImageStreamWithPreview(nodeName, streamer.Stream())

	nodeName += ":OUT"
	previewViewer := NewImageStreamViewer(
		nodeName,
		wrapped.Preview(),
		previewContainer,
		previewObjIdx,
		previewParams,
	)

	return wrapped, previewViewer, streamer
}
