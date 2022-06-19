package main

import (
	"context"

	"github.com/pkg/errors"
)

type Node interface {
	Name() string
	Run(context.Context)
	Err() <-chan error
}

type SourceNode[T any] struct {
	name     string
	outChan  chan T
	errChan  chan error
	setup    func() error
	teardown func() error
	step     func() (T, error)
}

var _ Node = &SourceNode[int]{}

func NewSourceNode[T any](name string) *SourceNode[T] {
	return &SourceNode[T]{
		name:     name,
		outChan:  make(chan T),
		errChan:  make(chan error),
		setup:    nil, // set by SetupFunc()
		teardown: nil, // set by TeardownFunc()
		step:     nil, // set by StepFunc()
	}
}

func (n *SourceNode[T]) Name() string {
	return n.name
}

func (n *SourceNode[T]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *SourceNode[T]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
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
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

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
	name     string
	inChan   <-chan T
	errChan  chan error
	setup    func() error
	teardown func() error
	step     func(T) error
}

var _ Node = &SinkNode[int]{}

func NewSinkNode[T any](name string, inChan <-chan T) *SinkNode[T] {
	return &SinkNode[T]{
		name:     name,
		inChan:   inChan,
		errChan:  make(chan error),
		setup:    nil, // set by SetupFunc()
		teardown: nil, // set by TeardownFunc()
		step:     nil, // set by StepFunc()
	}
}

func (n *SinkNode[T]) Name() string {
	return n.name
}

func (n *SinkNode[T]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *SinkNode[T]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
}

func (n *SinkNode[T]) StepFunc(step func(T) error) {
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
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

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
	name     string
	inChan   <-chan T
	outChan  chan T
	errChan  chan error
	setup    func() error
	teardown func() error
	step     func(T) (T, error)
}

var _ Node = &TransformerNode[int]{}

func NewTransformerNode[T any](name string, inChan <-chan T) *TransformerNode[T] {
	return &TransformerNode[T]{
		name:    name,
		inChan:  inChan,
		outChan: make(chan T),
		errChan: make(chan error),
		step:    nil, // set by StepFunc()
	}
}

func (n *TransformerNode[T]) Name() string {
	return n.name
}

func (n *TransformerNode[T]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *TransformerNode[T]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
}

func (n *TransformerNode[T]) StepFunc(step func(T) (T, error)) {
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
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

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
	name     string
	inChan   <-chan From
	outChan  chan To
	errChan  chan error
	setup    func() error
	teardown func() error
	step     func(From) (To, error)
}

var _ Node = &ConverterNode[int, int]{}

func NewConverterNode[From, To any](name string, inChan <-chan From) *ConverterNode[From, To] {
	return &ConverterNode[From, To]{
		name:     name,
		inChan:   inChan,
		outChan:  make(chan To),
		errChan:  make(chan error),
		setup:    nil, // set by SetupFunc()
		teardown: nil, // set by TeardownFunc()
		step:     nil, // set by StepFunc()
	}
}

func (n *ConverterNode[From, To]) Name() string {
	return n.name
}

func (n *ConverterNode[From, To]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *ConverterNode[From, To]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
}

func (n *ConverterNode[From, To]) StepFunc(step func(From) (To, error)) {
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
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

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

type CombinerNode[A, B, C any] struct {
	name     string
	inChanA  <-chan A
	inChanB  <-chan B
	outChan  chan C
	errChan  chan error
	setup    func() error
	teardown func() error
	step     func(A, B) (C, error)
}

var _ Node = &CombinerNode[bool, int, string]{}

func NewCombinerNode[A, B, C any](name string, inChanA <-chan A, inChanB <-chan B) *CombinerNode[A, B, C] {
	return &CombinerNode[A, B, C]{
		name:     name,
		inChanA:  inChanA,
		inChanB:  inChanB,
		outChan:  make(chan C),
		errChan:  make(chan error),
		setup:    nil, // set by SetupFunc()
		teardown: nil, // set by TeardownFunc()
		step:     nil, // set by StepFunc()
	}
}

func (n *CombinerNode[A, B, C]) Name() string {
	return n.name
}

func (n *CombinerNode[A, B, C]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *CombinerNode[A, B, C]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
}

func (n *CombinerNode[A, B, C]) StepFunc(step func(A, B) (C, error)) {
	n.step = step
}

func (n *CombinerNode[A, B, C]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *CombinerNode[A, B, C]) Err() <-chan error {
	return n.errChan
}

func (n *CombinerNode[A, B, C]) Stream() <-chan C {
	return n.outChan
}

func (n *CombinerNode[A, B, C]) loop(ctx context.Context) {
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

	for {
		a, err := BlockingRecv(ctx, n.inChanA)
		if err != nil {
			n.errChan <- err
			return
		}

		b, err := BlockingRecv(ctx, n.inChanB)
		if err != nil {
			n.errChan <- err
			return
		}

		c, err := n.step(a, b)
		if err != nil {
			n.errChan <- err
			return
		}

		err = BlockingSend(ctx, n.outChan, c)
		if err != nil {
			n.errChan <- err
			return
		}
	}
}

type PreviewerNode[T any, P any] struct {
	name        string
	inChan      <-chan T
	previewChan chan P
	outChan     chan T
	errChan     chan error
	setup       func() error
	teardown    func() error
	step        func(T) (P, error)
}

var _ Node = &PreviewerNode[int, bool]{}

func NewPreviewerNode[T any, P any](name string, inChan <-chan T) *PreviewerNode[T, P] {
	return &PreviewerNode[T, P]{
		name:        name,
		inChan:      inChan,
		previewChan: make(chan P),
		outChan:     make(chan T),
		errChan:     make(chan error),
		step:        nil, // set by StepFunc()
	}
}

func (n *PreviewerNode[T, P]) Name() string {
	return n.name
}

func (n *PreviewerNode[T, P]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *PreviewerNode[T, P]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
}

func (n *PreviewerNode[T, P]) StepFunc(step func(T) (P, error)) {
	n.step = step
}

func (n *PreviewerNode[T, P]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *PreviewerNode[T, P]) Err() <-chan error {
	return n.errChan
}

func (n *PreviewerNode[T, P]) Stream() <-chan T {
	return n.outChan
}

func (n *PreviewerNode[T, P]) Preview() <-chan P {
	return n.previewChan
}

func (n *PreviewerNode[T, P]) loop(ctx context.Context) {
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

	var (
		origValue    T
		previewValue P
		err          error
	)

	for {
		origValue, err = BlockingRecv(ctx, n.inChan)
		if err != nil {
			n.errChan <- err
			return
		}

		// Send the original value forward before generating the preview, so the nodes along the main stream
		// can continue working.
		err = BlockingSend(ctx, n.outChan, origValue)
		if err != nil {
			n.errChan <- err
			return
		}

		previewValue, err = n.step(origValue)
		if err != nil {
			n.errChan <- err
			return
		}

		err = NonBlockingSend(ctx, n.previewChan, previewValue)
		if err != nil {
			// It's OK if the preview is not sent.
			if errors.Is(err, NotSent) {
				continue
			}

			n.errChan <- err
			return
		}
	}
}

type ClonerNode[T any] struct {
	name      string
	inChan    <-chan T
	cloneChan chan T
	outChan   chan T
	errChan   chan error
	setup     func() error
	teardown  func() error
	step      func(T) (T, error)
}

var _ Node = &ClonerNode[int]{}

func NewClonerNode[T any](name string, inChan <-chan T) *ClonerNode[T] {
	return &ClonerNode[T]{
		name:      name,
		inChan:    inChan,
		cloneChan: make(chan T),
		outChan:   make(chan T),
		errChan:   make(chan error),
		step:      nil, // set by StepFunc()
	}
}

func (n *ClonerNode[T]) Name() string {
	return n.name
}

func (n *ClonerNode[T]) SetupFunc(setup func() error) {
	n.setup = setup
}

func (n *ClonerNode[T]) TeardownFunc(teardown func() error) {
	n.teardown = teardown
}

func (n *ClonerNode[T]) StepFunc(step func(T) (T, error)) {
	n.step = step
}

func (n *ClonerNode[T]) Run(ctx context.Context) {
	if n.step == nil {
		return
	}

	go n.loop(ctx)
}

func (n *ClonerNode[T]) Err() <-chan error {
	return n.errChan
}

func (n *ClonerNode[T]) Stream() <-chan T {
	return n.outChan
}

func (n *ClonerNode[T]) Clone() <-chan T {
	return n.cloneChan
}

func (n *ClonerNode[T]) loop(ctx context.Context) {
	if n.setup != nil {
		err := n.setup()
		if err != nil {
			n.errChan <- errors.Wrap(err, "setup error")
			return
		}
	}

	defer func() {
		if n.teardown != nil {
			err := n.teardown()
			if err != nil {
				n.errChan <- errors.Wrap(err, "teardown error")
			}
		}
	}()

	var (
		origValue  T
		cloneValue T
		err        error
	)

	for {
		origValue, err = BlockingRecv(ctx, n.inChan)
		if err != nil {
			n.errChan <- err
			return
		}

		cloneValue, err = n.step(origValue)
		if err != nil {
			n.errChan <- err
			return
		}

		err = BlockingSend(ctx, n.outChan, origValue)
		if err != nil {
			n.errChan <- err
			return
		}

		err = BlockingSend(ctx, n.cloneChan, cloneValue)
		if err != nil {
			n.errChan <- err
			return
		}
	}
}
