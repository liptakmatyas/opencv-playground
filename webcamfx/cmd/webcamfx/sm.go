package main

import (
	"context"
	"sort"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type State string

const (
	StateNone    State = ""
	StateFailed  State = "FAILED"
	StateStopped State = "STOPPED"
	StatePaused  State = "PAUSED"
	StateRunning State = "RUNNING"
)

type StateList []State

func (sl StateList) Sort() {
	sort.Slice(sl, func(i, j int) bool {
		return sl[i] < sl[j]
	})
}

type StateTransition func(context.Context, chan *stateChangeReq) (State, error)
type StateTransitionMap map[State]StateTransition

type smId string

type childErr struct {
	cid smId
	err error
}

type StateMachine struct {
	id smId

	currentStateMu sync.RWMutex
	currentState   State

	stepFrom     StateTransitionMap
	setStateChan chan *stateChangeReq

	children        map[smId]*StateMachine
	childrenErrChan chan childErr
}

var NoChildren map[smId]*StateMachine

func NewStateMachine(id smId, startState State, stateTransitions StateTransitionMap, children map[smId]*StateMachine) *StateMachine {
	sm := &StateMachine{
		id:              id,
		currentState:    startState,
		stepFrom:        stateTransitions,
		setStateChan:    make(chan *stateChangeReq),
		children:        children,
		childrenErrChan: make(chan childErr, len(children)),
	}

	return sm
}

func (sm *StateMachine) logger() *logrus.Entry {
	return logger.
		WithField("smid", sm.id)
}

func (sm *StateMachine) AllStates() StateList {
	allStateNames := make(StateList, 0, len(sm.stepFrom))
	for state, _ := range sm.stepFrom {
		allStateNames = append(allStateNames, state)
	}
	allStateNames.Sort()

	return allStateNames
}

func (sm *StateMachine) CurrentState() State {
	sm.currentStateMu.RLock()
	defer sm.currentStateMu.RUnlock()

	return sm.currentState
}

type stateChangeReq struct {
	newState State
	errChan  chan error
}

func (sm *StateMachine) knownState(state State) error {
	if state == StateStopped || state == StateFailed {
		return nil
	}

	if _, ok := sm.stepFrom[state]; !ok {
		return errors.Errorf("Unknown state '%s'.", state)
	}

	return nil
}

func (sm *StateMachine) SetState(newState State) error {
	if err := sm.knownState(newState); err != nil {
		return errors.Wrapf(err, "State not changed:")
	}

	req := &stateChangeReq{
		newState: newState,
		errChan:  make(chan error),
	}

	sm.setStateChan <- req
	return <-req.errChan
}

func (sm *StateMachine) setState(newState State) error {
	sm.currentStateMu.Lock()
	defer sm.currentStateMu.Unlock()

	sm.currentState = newState
	return sm.setChildrenState(newState)
}

func (sm *StateMachine) setChildrenState(newState State) error {
	if len(sm.children) < 1 {
		return nil
	}

	cherrChan := make(chan error, len(sm.children))
	for cid, child := range sm.children {
		cid, child := cid, child
		chLogger := sm.logger().WithField("cid", cid)

		go func() {
			cherr := child.SetState(newState)
			if cherr != nil {
				chLogger.WithError(cherr).Error("Child state change failed.")
			}
			cherrChan <- cherr
		}()
	}

	// FIXME: Return some aggregated error, not just the last one.
	var err error
	for i := 0; i < len(sm.children); i++ {
		err = <-cherrChan
	}

	return err
}

func (sm *StateMachine) Run(ctx context.Context) error {
	sm.runChildren(ctx)
	return sm.loop(ctx)
}

func (sm *StateMachine) runChildren(ctx context.Context) {
	for cid, child := range sm.children {
		cid, child := cid, child
		chLogger := sm.logger().WithField("cid", cid)

		go func() {
			err := child.Run(ctx)
			if err != nil {
				chLogger.WithError(err).Error("Child run failed.")
			} else {
				chLogger.Info("Child run succeeded.")
			}

			// Send error to loop().
			chLogger.Trace("pre-send")
			sm.childrenErrChan <- childErr{cid, err}
			chLogger.Trace("post-send")
		}()
	}
}

func (sm *StateMachine) loop(ctx context.Context) (loopErr error) {
	loopErr = nil
	defer func() {
		if loopErr != nil {
			err := sm.setState(StateFailed)
			if err != nil {
				loopErr = errors.Wrapf(err, "while processing: %v", loopErr)
			}

			return
		}

		err := sm.setState(StateStopped)
		if err != nil {
			loopErr = err
			return
		}
	}()

	for {
		state := sm.CurrentState()

		if state == StateStopped || state == StateFailed {
			return
		}

		stepFunc, ok := sm.stepFrom[state]
		if !ok {
			return errors.Errorf("No transition from state '%s'.", state)
		}

		newState, err := stepFunc(ctx, sm.setStateChan)
		sm.logger().Debugf("newState=%s, err=%v", newState, err)
		if err != nil {
			return errors.Wrapf(err, "Transition from state '%s' failed.", state)
		}

		err = sm.setState(newState)
		if err != nil {
			return errors.Wrapf(err,
				"Could not set new state '%s' after transition from state '%s'",
				newState, state)
		}

		select {
		case <-ctx.Done():
			return

		case req := <-sm.setStateChan:
			sm.logger().Tracef("setting newState: %s", req.newState)
			req.errChan <- sm.setState(req.newState)

		case cherr := <-sm.childrenErrChan:
			if cherr.err != nil {
				return cherr.err
			}

		default:
			continue
		}
	}
}

func pausedState(ctx context.Context, stateChan chan *stateChangeReq) (State, error) {
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

//pausedState = func(play chan struct{}) StateTransition {
//	logger.Tracef("Entered PAUSED state.")
//	defer logger.Tracef("Leaving PAUSED state.")
//
//	<-play
//	return playingState
//}

//playingState = func(pause chan struct{}) StateTransition {
//	logger.Tracef("Entered PLAYING state.")
//	defer logger.Tracef("Leaving PLAYING state.")
//
//	logger.Infof("Opening video capture source.")
//	webcam, err := gocv.OpenVideoCapture(sourceId)
//	if err != nil {
//		logger.WithError(err).Errorf("Error opening video capture source.")
//		return pausedState
//	}
//	defer func() {
//		logger.Infof("Closing video capture source.")
//		_ = webcam.Close()
//	}()
//
//	buf := gocv.NewMat()
//	defer func() { _ = buf.Close() }()
//
//	logger.Infof("Start reading video capture source.")
//	for {
//		if ok := webcam.Read(&buf); !ok {
//			logger.Warnf("Video capture source could not read raw frame.")
//			return pausedState
//		}
//
//		if buf.Empty() {
//			logger.Warnf("Captured empty frame.")
//			continue
//		}
//
//		img, err := buf.ToImage()
//		if err != nil {
//			logger.
//				WithError(err).
//				Warnf("Failed to convert raw frame to image.")
//			continue
//		}
//
//		select {
//		case <-pause:
//			return pausedState
//
//		case rawFrames <- img:
//			continue
//		}
//	}
//}
