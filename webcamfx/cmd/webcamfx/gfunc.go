package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

func gfuncMain(parentCtx context.Context) error {
	ctx, cancelCtx := context.WithCancel(parentCtx)

	src := NewSourceNode[int]("RandInt")
	src.StepFunc(func() (int, error) {
		v := rand.Int()
		fmt.Printf("Source generated new value: %+v\n", v)
		return v, nil
	})

	trf := NewTransformerNode[int]("Halver", src.Stream())
	trf.StepFunc(func(v int) (int, error) {
		fmt.Printf("Transformer received value: %+v\n", v)
		v = v / 2
		fmt.Printf("Transformer created new value: %+v\n", v)
		return v, nil
	})

	cnv := NewConverterNode[int, string]("Stringizer", trf.Stream())
	cnv.StepFunc(func(v int) (string, error) {
		fmt.Printf("Converter received value: %+v\n", v)
		v2 := fmt.Sprintf("v=%d", v)
		fmt.Printf("Converter created new value: %+v\n", v2)
		return v2, nil
	})

	sink := NewSinkNode[string]("Printer", cnv.Stream())
	sink.StepFunc(func(v string) error {
		fmt.Printf("Sink received value: %+v\n", v)
		return nil
	})

	g := NewGraph("IntStreamer")
	g.SetNodes(src, trf, cnv, sink)
	g.Run(ctx)

	timeout := time.NewTimer(1000 * time.Millisecond)
	for {
		select {
		case err := <-g.Err():
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			return err

		case <-timeout.C:
			fmt.Printf("TIMEOUT REACHED\n")
			cancelCtx()
		}
	}
}
