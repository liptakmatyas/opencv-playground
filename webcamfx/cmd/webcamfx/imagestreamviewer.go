package main

import (
	"context"
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

type ImageStream struct {
	id smId

	sourceId      string
	rawFramesChan chan<- image.Image
	sm            *StateMachine
}

func NewImageStream(id smId, sourceId string, rawFramesChan chan<- image.Image) *ImageStream {
	is := &ImageStream{
		id:            id,
		sourceId:      sourceId,
		rawFramesChan: rawFramesChan,
	}

	smid := smId(fmt.Sprintf("%s:sm", is.id))
	is.sm = NewStateMachine(smid, StatePaused, StateTransitionMap{
		StatePaused:  is.pausedState,
		StateRunning: is.runningState,
	}, NoChildren)

	return is
}

func (is *ImageStream) logger() *logrus.Entry {
	return logger.
		WithField("Component", "ImageStream").
		WithField("id", is.id).
		WithField("SourceId", is.sourceId).
		WithField("sm.id", is.sm.id).
		WithField("sm.State", is.sm.CurrentState())
}

func (is *ImageStream) pausedState(ctx context.Context, stateChan chan *stateChangeReq) (State, error) {
	logger := is.logger
	logger().Tracef("Entered state.")
	defer logger().Tracef("Leaving state.")

	for {
		select {
		case <-ctx.Done():
			return StateStopped, nil

		case req := <-stateChan:
			req.errChan <- nil
			return req.newState, nil
		}

	}
}

func (is *ImageStream) runningStateSetup() (*gocv.VideoCapture, gocv.Mat, error) {
	logger := is.logger
	logger().Infof("Opening video capture source.")
	webcam, err := gocv.OpenVideoCapture(is.sourceId)
	if err != nil {
		logger().WithError(err).Errorf("Error opening video capture source.")
		return nil, gocv.Mat{}, err
	}

	buf := gocv.NewMat()

	return webcam, buf, nil
}

func (is *ImageStream) runningState(ctx context.Context, stateChan chan *stateChangeReq) (State, error) {
	logger := is.logger
	logger().Tracef("Entered state.")
	defer logger().Tracef("Leaving state.")

	webcam, buf, err := is.runningStateSetup()
	if err != nil {
		return StateFailed, err
	}
	defer func() {
		logger().Warnf("Closing video capture source.")
		_ = webcam.Close()
		_ = buf.Close()
	}()

	for {
		// Receive signals and commands, if any.

		select {
		case <-ctx.Done():
			return StateStopped, nil

		case req := <-stateChan:
			req.errChan <- nil
			return req.newState, nil

		default:
			break // from select
		}

		// Produce new frame.

		if ok := webcam.Read(&buf); !ok {
			logger().Warnf("Video capture source could not read raw frame.")
			return StateStopped, nil
		}

		if buf.Empty() {
			logger().Warnf("Captured empty frame.")
			continue // loop
		}

		newFrame, err := buf.ToImage()
		if err != nil {
			logger().
				WithError(err).
				Error("Failed to convert raw frame to image.")
			continue // loop
		}

		// Send produced frame.

		select {
		case <-ctx.Done():
			return StateStopped, nil

		case is.rawFramesChan <- newFrame:
			continue // loop

		default:
			continue // loop
		}
	}
}

type ImageStreamViewer struct {
	id smId

	toolbar         *widget.Toolbar
	playPauseButton *widget.ToolbarAction
	container       *fyne.Container

	imageStream   *ImageStream
	rawFramesChan <-chan image.Image
	sm            *StateMachine
}

func NewImageStreamViewer(
	id smId,
	toolbar *widget.Toolbar,
	is *ImageStream,
	rawFramesChan <-chan image.Image,
) *ImageStreamViewer {

	isv := &ImageStreamViewer{
		id:            id,
		toolbar:       toolbar,
		imageStream:   is,
		rawFramesChan: rawFramesChan,
	}

	isvSmId := smId(fmt.Sprintf("%s:sm", id))
	isv.sm = NewStateMachine(isvSmId, StatePaused, StateTransitionMap{
		StatePaused:  isv.pausedState,
		StateRunning: isv.runningState,
	}, map[smId]*StateMachine{
		is.id: is.sm,
	})

	isv.playPauseButton = widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
		nextState := StateRunning
		if isv.sm.CurrentState() == StateRunning {
			nextState = StatePaused
		}
		isv.sm.SetState(nextState)
	}).(*widget.ToolbarAction)
	isv.toolbar.Append(isv.playPauseButton)

	rawFrameView := canvas.NewImageFromResource(theme.MediaVideoIcon())
	rawFrameView.SetMinSize(fyne.NewSize(640.0, 480.0))
	isv.container = container.New(layout.NewCenterLayout(), rawFrameView)

	return isv
}

func (isv *ImageStreamViewer) Container() *fyne.Container {
	return isv.container
}

func (isv *ImageStreamViewer) logger() *logrus.Entry {
	return logger.
		WithField("Component", "ImageStreamViewer").
		WithField("id", isv.id).
		WithField("sm.id", isv.sm.id).
		WithField("sm.State", isv.sm.CurrentState())
}

func (isv *ImageStreamViewer) pausedState(ctx context.Context, stateChan chan *stateChangeReq) (s State, err error) {
	logger := isv.logger
	logger().Tracef("Entered state.")
	defer func() { logger().Tracef("Leaving state. s=%s", s) }()

	isv.playPauseButton.Icon = theme.MediaPlayIcon()
	isv.toolbar.Refresh()

	for {
		select {
		case <-ctx.Done():
			return StateStopped, nil

		case req := <-stateChan:
			logger().Tracef("newState: %s", req.newState)
			req.errChan <- nil
			return req.newState, nil
		}
	}
}

func (isv *ImageStreamViewer) runningState(ctx context.Context, stateChan chan *stateChangeReq) (State, error) {
	logger := isv.logger
	logger().Tracef("Entered state.")
	defer logger().Tracef("Leaving state.")

	isv.playPauseButton.Icon = theme.MediaPauseIcon()
	isv.toolbar.Refresh()

	for {
		select {
		case <-ctx.Done():
			return StateStopped, nil

		case req := <-stateChan:
			req.errChan <- nil
			return req.newState, nil

		case rawFrame := <-isv.rawFramesChan:
			logger().Tracef("Received raw frame")
			rawFrameBounds := rawFrame.Bounds()

			img := canvas.NewImageFromImage(rawFrame)
			img.SetMinSize(fyne.NewSize(float32(rawFrameBounds.Dx()), float32(rawFrameBounds.Dy())))
			img.FillMode = canvas.ImageFillOriginal

			// we assume the first object in the container is the raw frame view
			isv.container.Objects[0] = img
			isv.container.Refresh()
		}
	}
}

//func makeVideoCaptureLoop(sourceId string, rawFrames chan image.Image) (chan struct{}, func()) {
//	logger := logger.
//		WithField("func", "makeVideoCaptureLoop").
//		WithField("SourceId", sourceId)
//
//	type stateTransition func(chan struct{}) stateTransition
//	var (
//		state        stateTransition
//		pausedState  stateTransition
//		playingState stateTransition
//	)
//
//	playPauseChan := make(chan struct{})
//	stateMachineLoop := func() {
//		logger.Tracef("Loop started.")
//		defer logger.Tracef("Loop stopped.")
//
//		state = pausedState
//		for state != nil {
//			state = state(playPauseChan)
//		}
//	}
//
//	pausedState = func(play chan struct{}) stateTransition {
//		logger.Tracef("Entered PAUSED state.")
//		defer logger.Tracef("Leaving PAUSED state.")
//
//		<-play
//		return playingState
//	}
//
//	playingState = func(pause chan struct{}) stateTransition {
//		logger.Tracef("Entered PLAYING state.")
//		defer logger.Tracef("Leaving PLAYING state.")
//
//		logger.Infof("Opening video capture source.")
//		webcam, err := gocv.OpenVideoCapture(sourceId)
//		if err != nil {
//			logger.WithError(err).Errorf("Error opening video capture source.")
//			return pausedState
//		}
//		defer func() {
//			logger.Infof("Closing video capture source.")
//			_ = webcam.Close()
//		}()
//
//		buf := gocv.NewMat()
//		defer func() { _ = buf.Close() }()
//
//		logger.Infof("Start reading video capture source.")
//		for {
//			if ok := webcam.Read(&buf); !ok {
//				logger.Warnf("Video capture source could not read raw frame.")
//				return pausedState
//			}
//
//			if buf.Empty() {
//				logger.Warnf("Captured empty frame.")
//				continue
//			}
//
//			img, err := buf.ToImage()
//			if err != nil {
//				logger.
//					WithError(err).
//					Warnf("Failed to convert raw frame to image.")
//				continue
//			}
//
//			select {
//			case <-pause:
//				return pausedState
//
//			case rawFrames <- img:
//				continue
//			}
//		}
//	}
//
//	return playPauseChan, stateMachineLoop
//}
//
//func makeRawFrameViewLoop(args *CliArgs, toolbar *widget.Toolbar, container *fyne.Container) (widget.ToolbarItem, func()) {
//	logger := logger.
//		WithField("func", "makeRawFrameViewLoop")
//
//	rawFrames := make(chan image.Image)
//	capturePlayPauseChan, captureLoop := makeVideoCaptureLoop(args.SourceId, rawFrames)
//
//	type stateTransition func(chan struct{}) stateTransition
//	var (
//		state        stateTransition
//		pausedState  stateTransition
//		playingState stateTransition
//	)
//
//	playPauseChan := make(chan struct{})
//	stateMachineLoop := func() {
//		logger.Tracef("Loop started.")
//		defer logger.Tracef("Loop stopped.")
//
//		go captureLoop()
//
//		state = pausedState
//		for {
//			state = state(playPauseChan)
//		}
//	}
//
//	playPauseButton := widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
//		playPauseChan <- struct{}{}
//	}).(*widget.ToolbarAction)
//
//	pausedState = func(play chan struct{}) stateTransition {
//		logger.Tracef("Entered PAUSED state.")
//		defer logger.Tracef("Leaving PAUSED state.")
//
//		playPauseButton.Icon = theme.MediaPlayIcon()
//		toolbar.Refresh()
//
//		<-play
//		capturePlayPauseChan <- struct{}{}
//		return playingState
//	}
//
//	playingState = func(pause chan struct{}) stateTransition {
//		logger.Tracef("Entered PLAYING state.")
//		defer logger.Tracef("Leaving PLAYING state.")
//
//		playPauseButton.Icon = theme.MediaPauseIcon()
//		toolbar.Refresh()
//
//		for {
//			select {
//			case <-pause:
//				capturePlayPauseChan <- struct{}{}
//				return pausedState
//
//			case rawFrame := <-rawFrames:
//				logger.Tracef("Received raw frame")
//				rawFrameBounds := rawFrame.Bounds()
//
//				img := canvas.NewImageFromImage(rawFrame)
//				img.SetMinSize(fyne.NewSize(float32(rawFrameBounds.Dx()), float32(rawFrameBounds.Dy())))
//				img.FillMode = canvas.ImageFillOriginal
//
//				// we assume the first object in the container is the raw frame view
//				container.Objects[0] = img
//				container.Refresh()
//			}
//		}
//	}
//
//	return playPauseButton, stateMachineLoop
//}
