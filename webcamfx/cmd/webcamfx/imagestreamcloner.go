package main

import (
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

type ImageStreamCloner struct {
	*ClonerNode[*gocv.Mat]
}

var _ Node = &ImageStreamCloner{}

func NewImageStreamCloner(name string, inChan <-chan *gocv.Mat) *ImageStreamCloner {
	isc := &ImageStreamCloner{
		ClonerNode: NewClonerNode[*gocv.Mat](name, inChan),
	}

	var (
		cloneBuffer gocv.Mat
	)

	isc.SetupFunc(func() error {
		cloneBuffer = gocv.NewMat()
		return nil
	})

	isc.TeardownFunc(func() error {
		return errors.Wrapf(cloneBuffer.Close(), "clone buffer teardown error")
	})

	isc.StepFunc(func(img *gocv.Mat) (*gocv.Mat, error) {
		if img.Empty() {
			return nil, nil
		}

		img.CopyTo(&cloneBuffer)
		return &cloneBuffer, nil
	})

	return isc
}
