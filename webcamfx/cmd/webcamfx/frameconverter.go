package main

import (
	"image"

	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type FrameConverter struct {
	*ConverterNode[*gocv.Mat, image.Image]
}

var _ Node = &FrameConverter{}

func NewFrameConverter(name string, inChan <-chan *gocv.Mat) *FrameConverter {
	fc := &FrameConverter{
		ConverterNode: NewConverterNode[*gocv.Mat, image.Image](name, inChan),
	}

	fc.StepFunc(func(rawFrame *gocv.Mat) (image.Image, error) {
		if rawFrame == nil {
			return nil, nil
		}

		img, err := rawFrame.ToImage()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert raw frame")
		}

		return img, err
	})

	return fc
}
