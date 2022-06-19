package main

import (
	"context"
)

type Node interface {
	Name() string
	Run(context.Context)
	Err() <-chan error
}

type SourceNode[T any] struct {
	name    string
	outChan chan T
	errChan chan error
	step    func() (T, error)
}

var _ Node = &SourceNode[int]{}

func NewSourceNode[T any](name string) *SourceNode[T] {
	return &SourceNode[T]{
		name:    name,
		outChan: make(chan T),
		errChan: make(chan error),
		step:    nil, // will be set in StepFunc()
	}
}

func (n *SourceNode[T]) Name() string {
	return n.name
}

func (n *SourceNode[T]) StepFunc(step func() (T, error)) {
	n.step = step
}

func (n *SourceNode[T]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *SourceNode[T]) Err() <-chan error {
	return n.errChan
}

func (n *SourceNode[T]) Stream() <-chan T {
	return n.outChan
}

func (n *SourceNode[T]) loop(ctx context.Context) {
	for {
		v, err := n.step()
		if err != nil {
			n.errChan <- err
			return
		}

		err = BlockingSend(ctx, n.outChan, v)
		if err != nil {
			n.errChan <- err
			return
		}
	}
}

type SinkNode[T any] struct {
	name    string
	inChan  <-chan T
	errChan chan error
	step    func(T) error
}

var _ Node = &SinkNode[int]{}

func NewSinkNode[T any](name string, inChan <-chan T) *SinkNode[T] {
	return &SinkNode[T]{
		name:    name,
		inChan:  inChan,
		errChan: make(chan error),
		step:    nil, // will be set in StepFunc()
	}
}

func (n *SinkNode[T]) Name() string {
	return n.name
}

func (n *SinkNode[T]) StepFunc(step func(v T) error) {
	n.step = step
}

func (n *SinkNode[T]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *SinkNode[T]) Err() <-chan error {
	return n.errChan
}

func (n *SinkNode[T]) loop(ctx context.Context) {
	for {
		v, err := BlockingRecv(ctx, n.inChan)
		if err != nil {
			n.errChan <- err
			return
		}

		err = n.step(v)
		if err != nil {
			n.errChan <- err
			return
		}
	}
}

type TransformerNode[T any] struct {
	name    string
	inChan  <-chan T
	outChan chan T
	errChan chan error
	step    func(T) (T, error)
}

var _ Node = &TransformerNode[int]{}

func NewTransformerNode[T any](name string, inChan <-chan T) *TransformerNode[T] {
	return &TransformerNode[T]{
		name:    name,
		inChan:  inChan,
		outChan: make(chan T),
		errChan: make(chan error),
		step:    nil, // will be set in StepFunc()
	}
}

func (n *TransformerNode[T]) Name() string {
	return n.name
}

func (n *TransformerNode[T]) StepFunc(step func(v T) (T, error)) {
	n.step = step
}

func (n *TransformerNode[T]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *TransformerNode[T]) Err() <-chan error {
	return n.errChan
}

func (n *TransformerNode[T]) Stream() <-chan T {
	return n.outChan
}

func (n *TransformerNode[T]) loop(ctx context.Context) {
	for {
		v, err := BlockingRecv(ctx, n.inChan)
		if err != nil {
			n.errChan <- err
			return
		}

		v2, err := n.step(v)
		if err != nil {
			n.errChan <- err
			return
		}

		err = BlockingSend(ctx, n.outChan, v2)
		if err != nil {
			n.errChan <- err
			return
		}
	}
}

type ConverterNode[From, To any] struct {
	name    string
	inChan  <-chan From
	outChan chan To
	errChan chan error
	step    func(From) (To, error)
}

var _ Node = &ConverterNode[int, int]{}

func NewConverterNode[From, To any](name string, inChan <-chan From) *ConverterNode[From, To] {
	return &ConverterNode[From, To]{
		name:    name,
		inChan:  inChan,
		outChan: make(chan To),
		errChan: make(chan error),
		step:    nil, // will be set in StepFunc()
	}
}

func (n *ConverterNode[From, To]) Name() string {
	return n.name
}

func (n *ConverterNode[From, To]) StepFunc(step func(v From) (To, error)) {
	n.step = step
}

func (n *ConverterNode[From, To]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *ConverterNode[From, To]) Err() <-chan error {
	return n.errChan
}

func (n *ConverterNode[From, To]) Stream() <-chan To {
	return n.outChan
}

func (n *ConverterNode[From, To]) loop(ctx context.Context) {
	for {
		v, err := BlockingRecv(ctx, n.inChan)
		if err != nil {
			n.errChan <- err
			return
		}

		v2, err := n.step(v)
		if err != nil {
			n.errChan <- err
			return
		}

		err = BlockingSend(ctx, n.outChan, v2)
		if err != nil {
			n.errChan <- err
			return
		}
	}
}
