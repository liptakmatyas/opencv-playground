package main

import (
	"context"

	"github.com/pkg/errors"
)

var (
	NotSent         = errors.New("value not sent")
	ReceivedNothing = errors.New("received nothing")
)

func BlockingSend[T any](ctx context.Context, sendChan chan<- T, sendValue T) error {
	select {
	case <-ctx.Done():
		return context.Canceled

	case sendChan <- sendValue:
		return nil
	}
}

func NonBlockingSend[T any](ctx context.Context, sendChan chan<- T, sendValue T) error {
	select {
	case <-ctx.Done():
		return context.Canceled

	case sendChan <- sendValue:
		return nil

	default:
		return NotSent
	}
}

func BlockingRecv[T any](ctx context.Context, recvChan <-chan T) (T, error) {
	var recvValue T

	select {
	case <-ctx.Done():
		return recvValue, context.Canceled

	case recvValue = <-recvChan:
		return recvValue, nil
	}
}

func NonBlockingRecv[T any](ctx context.Context, recvChan <-chan T) (T, error) {
	var recvValue T

	select {
	case <-ctx.Done():
		return recvValue, context.Canceled

	case recvValue = <-recvChan:
		return recvValue, nil

	default:
		return recvValue, ReceivedNothing
	}
}
