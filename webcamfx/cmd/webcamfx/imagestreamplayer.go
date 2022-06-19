package main

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

type ImageStreamPlayer struct {
	source FrameStreamerNode

	toolbar *widget.Toolbar
	action  *widget.ToolbarAction

	imgViewContainer *fyne.Container
	imgViewObjIdx    int

	settingsContainer           *fyne.Container
	vcsSettingsContainer        *fyne.Container
	vcsPreviewContainer         *fyne.Container
	vcsPreviewObjIdx            int
	vcsPreviewSettingsContainer *fyne.Container
	vcsPreviewParams            ImageStreamViewerParameters

	vsParams *VideoSourceParameters
	fxMakers fxMakerFuncs
	fxParams []SettingsContainerMaker

	states         map[state]stateDetails
	state          state
	stateChange    chan state
	cancelStateCtx context.CancelFunc
	stateErr       <-chan error
	errChan        chan error
}

type FrameStreamerNode interface {
	Node
	Stream() <-chan *gocv.Mat
}

type FrameStreamerWithPreviewNode interface {
	Node
	Stream() <-chan *gocv.Mat
	Preview() <-chan *gocv.Mat
}

type fxMakerFunc func(<-chan *gocv.Mat) FrameStreamerNode
type fxMakerFuncs []fxMakerFunc

type SettingsContainerMaker interface {
	MakeSettingsContainer(fyne.Window) *fyne.Container
}

type vsMakerFunc func(name string, vsp *VideoSourceParameters) FrameStreamerNode

func NewImageStreamPlayer(
	win fyne.Window,
	toolbar *widget.Toolbar,
	vsMaker vsMakerFunc,
	vsParams *VideoSourceParameters,
	fxMakers fxMakerFuncs,
	fxParams []SettingsContainerMaker,
) *ImageStreamPlayer {
	isp := &ImageStreamPlayer{
		source:                      nil, // set below
		toolbar:                     toolbar,
		imgViewContainer:            container.New(layout.NewCenterLayout(), DefaultNoSignalImage(fyne.NewSize(640.0, 480.0))),
		imgViewObjIdx:               0,   // added the default image above as the first in imgViewContainer
		settingsContainer:           nil, // set by makeSettingsContainer() below
		vcsSettingsContainer:        nil, // set by makeSettingsContainer() below
		vcsPreviewContainer:         nil, // set by makeSettingsContainer() below
		vcsPreviewObjIdx:            0,   // set by makeSettingsContainer() below
		vcsPreviewSettingsContainer: nil, // set by makeSettingsContainer() below
		vcsPreviewParams: ImageStreamViewerParameters{
			enabled: true,
		},

		vsParams: vsParams,
		fxMakers: fxMakers,
		fxParams: fxParams,

		state:       stopped,
		stateChange: make(chan state),
		stateErr:    nil, // set when state loop is started; nil means "not yet started"
		errChan:     make(chan error),
	}

	isp.source = vsMaker("VSRC", isp.vsParams)

	isp.states = map[state]stateDetails{
		stopped: {
			name:       "STOPPED",
			buttonIcon: theme.MediaPlayIcon(),
			stateLoop:  isp.stoppedState,
		},
		playing: {
			name:       "PLAYING",
			buttonIcon: theme.MediaStopIcon(),
			stateLoop:  isp.playingState,
		},
	}

	initialIcon := isp.states[isp.state].buttonIcon
	isp.action = widget.NewToolbarAction(initialIcon, isp.changeState).(*widget.ToolbarAction)

	isp.makeSettingsContainer(win)

	return isp
}

func (isp *ImageStreamPlayer) makeSettingsContainer(win fyne.Window) {
	isp.vcsPreviewContainer, isp.vcsPreviewObjIdx = MakePreviewContainer(win)
	isp.vcsPreviewSettingsContainer = isp.vcsPreviewParams.MakeSettingsContainer(win)
	isp.vcsSettingsContainer = container.New(layout.NewVBoxLayout(),
		isp.vsParams.MakeSettingsContainer(win),
		container.New(layout.NewGridLayout(2),
			isp.vcsPreviewSettingsContainer,
			isp.vcsPreviewContainer,
		),
	)

	isp.settingsContainer = container.New(layout.NewVBoxLayout(),
		widget.NewSeparator(),
		isp.vcsSettingsContainer,
		widget.NewSeparator(),
	)

	// Optional FX Parameters
	logrus.Debugf("ISP: fxParams=%v", isp.fxParams)
	if isp.fxParams != nil {
		for _, fxParam := range isp.fxParams {
			fxParam := fxParam
			isp.settingsContainer.Add(fxParam.MakeSettingsContainer(win))
			isp.settingsContainer.Add(widget.NewSeparator())
		}
	}
}

func (isp *ImageStreamPlayer) ToolbarAction() *widget.ToolbarAction {
	return isp.action
}

func (isp *ImageStreamPlayer) ViewContainer() *fyne.Container {
	return isp.imgViewContainer
}

func (isp *ImageStreamPlayer) SettingsContainer() *fyne.Container {
	return isp.settingsContainer
}

func (isp *ImageStreamPlayer) Run(ctx context.Context) {
	isp.runState()
	go isp.loop(ctx)
}

func (isp *ImageStreamPlayer) Err() <-chan error {
	return isp.errChan
}

func (isp *ImageStreamPlayer) loop(ctx context.Context) {
	logger := logger.
		WithField("widget", "ISP").
		WithField("state", isp.states[isp.state])

	for {
		select {
		case <-ctx.Done():
			isp.errChan <- nil
			return

		case newState := <-isp.stateChange:
			isp.switchState(newState)

		case err := <-isp.stateErr:
			if err != nil {
				logger.WithError(err).Tracef("State error.")
				isp.errChan <- err
				return
			}

			logger.Tracef("State terminated successfully.")
		}
	}
}

func (isp *ImageStreamPlayer) changeState() {
	var newState state
	switch isp.state {
	case stopped:
		newState = playing
	default:
		newState = stopped
	}

	isp.stateChange <- newState
}

func (isp *ImageStreamPlayer) switchState(newState state) {
	isp.cancelStateCtx()

	logger.Tracef("Switching from state '%s' to state '%s' ...",
		isp.states[isp.state].name, isp.states[newState].name)
	isp.state = newState

	isp.action.Icon = isp.states[isp.state].buttonIcon
	isp.toolbar.Refresh()

	isp.runState()
}

func (isp *ImageStreamPlayer) runState() {
	stateCtx, cancel := context.WithCancel(context.Background())
	isp.cancelStateCtx = cancel

	s := isp.states[isp.state]
	isp.stateErr = s.stateLoop(stateCtx)
}

func (isp *ImageStreamPlayer) stoppedState(ctx context.Context) <-chan error {
	isp.imgViewContainer.Objects[isp.imgViewObjIdx] = DefaultNoSignalImage(fyne.NewSize(640.0, 480.0))
	isp.imgViewContainer.Refresh()

	ec := make(chan error)
	go func() {
		<-ctx.Done()
		ec <- context.Canceled
	}()

	return ec
}

func (isp *ImageStreamPlayer) playingState(ctx context.Context) <-chan error {
	logger := logger.WithField("widget", "ISP")

	logger.Tracef("Creating xgraph ...")
	stream := NewGraph("ISP")

	vcs, vcsPreviewViewer, vcsOrig := WrapFrameStreamerWithPreview(
		isp.source,
		isp.vcsPreviewContainer,
		isp.vcsPreviewObjIdx,
		&isp.vcsPreviewParams,
	)
	stream.SetNode(vcs)
	stream.SetNode(vcsPreviewViewer)
	stream.SetNode(vcsOrig)

	frameStream := vcs.Stream()
	if isp.fxMakers != nil {
		for _, makeFx := range isp.fxMakers {
			makeFx := makeFx
			fx := makeFx(frameStream)
			logger.Infof("Adding FX %s ...", fx.Name())
			stream.SetNode(fx)
			frameStream = fx.Stream()
		}
	}

	cnv := NewFrameConverter("CNV", frameStream)
	stream.SetNode(cnv)

	view := NewImageStreamViewer("VIEW", cnv.Stream(), isp.imgViewContainer, isp.imgViewObjIdx, &ImageStreamViewerParameters{true})
	stream.SetNode(view)

	logger.Tracef("Starting xgraph ...")
	stream.Run(ctx)
	return stream.Err()
}

func MakePreviewContainer(_ fyne.Window) (*fyne.Container, int) {
	c := container.New(layout.NewCenterLayout(),
		DefaultNoSignalImage(fyne.NewSize(160, 120)),
	)

	return c, 0
}

func DefaultNoSignalImage(size fyne.Size) *canvas.Image {
	img := canvas.NewImageFromResource(theme.MediaVideoIcon())
	img.SetMinSize(size)

	return img
}

type state int8

const (
	stopped state = 0
	playing state = 1
)

type stateDetails struct {
	name       string
	buttonIcon fyne.Resource
	stateLoop  func(context.Context) <-chan error
}
