package main

import (
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

func clampFloat64ToUInt8(f float64) uint8 {
	return uint8(math.Max(0.0, math.Min(f, 255.0)))
}

func colorFromScalar(s gocv.Scalar) color.NRGBA {
	return color.NRGBA{
		R: clampFloat64ToUInt8(s.Val1),
		G: clampFloat64ToUInt8(s.Val2),
		B: clampFloat64ToUInt8(s.Val3),
		A: clampFloat64ToUInt8(s.Val4),
	}
}

func scalarFromColor(c color.Color) gocv.Scalar {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	s := gocv.Scalar{
		Val1: float64(rgba.B),
		Val2: float64(rgba.G),
		Val3: float64(rgba.R),
		Val4: float64(rgba.A),
	}
	logger.Debugf("[scalarFromColor] rgba=%T%+v | s=%v", rgba, rgba, s)
	return s
}
