package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

const AllImageOps = `^[edED]*$`

var opsByOpCode = map[byte]opFunc{
	'e': gocv.Erode,
	'd': gocv.Dilate,
	'E': nOps(10, gocv.Erode),
	'D': nOps(10, gocv.Dilate),
}

func UnknownImageOpErrMsg(knownOpsRegexp string) string {
	return fmt.Sprintf("imgop must match /%s/", knownOpsRegexp)
}

type opFunc func(src gocv.Mat, dst *gocv.Mat, kernel gocv.Mat)

func nOps(n int, op opFunc) opFunc {
	return func(src gocv.Mat, dst *gocv.Mat, kernel gocv.Mat) {
		for i := 0; i <= n; i++ {
			op(src, dst, kernel)
		}
	}
}

func runOp(opCode byte, img *gocv.Mat, kernel *gocv.Mat) {
	op, ok := opsByOpCode[opCode]
	if !ok {
		return // Unknown opCode is a noop.
	}

	op(*img, img, *kernel)
}

func runOps(ops string, img *gocv.Mat, kernel *gocv.Mat) {
	for _, opCode := range ops {
		runOp(byte(opCode), img, kernel)
	}
}
