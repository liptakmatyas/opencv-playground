package main

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var toStoppedTrFunc = func(_ context.Context, _ chan *stateChangeReq) (State, error) {
	return StateStopped, nil
}

var toPausedTrFunc = func(_ context.Context, _ chan *stateChangeReq) (State, error) {
	return StatePaused, nil
}

func holdStateForNStepsThenSwitch(holdState State, nSteps int, endState State) StateTransition {
	stepsLeft := nSteps

	return func(_ context.Context, _ chan *stateChangeReq) (State, error) {
		if stepsLeft > 0 {
			stepsLeft--
			return holdState, nil
		}

		return endState, nil
	}
}

var toRunningTrFunc = func(_ context.Context, _ chan *stateChangeReq) (State, error) {
	return StateRunning, nil
}

var failingTrFuncError = errors.New("test error message")
var failingTrFunc = func(_ context.Context, _ chan *stateChangeReq) (State, error) {
	return "SOME_BOGUS_STATE_THAT_SHOULD_NOT_MATTER", failingTrFuncError
}

type TestInput struct {
	SmId             smId
	StartState       State
	StateTransitions StateTransitionMap
}

type TestExpectation struct {
	StartState State
	FinalState State
	LoopErr    error
	AllStates  StateList
}

type TestCase struct {
	in *TestInput
	ex *TestExpectation
}

func NewTestCase(in *TestInput, ex *TestExpectation) *TestCase {
	tc := &TestCase{in, ex}
	tc.prepare()

	return tc
}

func (tc *TestCase) prepare() {
	tc.ex.AllStates.Sort()
}

type TestSet map[string]*TestCase

type testCaseFunc func(*require.Assertions, *testing.T, *TestCase)

func (ts TestSet) Run(assert *require.Assertions, t *testing.T, tcFunc testCaseFunc) {
	for testName, tc := range ts {
		t.Run(testName, func(t *testing.T) {
			tcFunc(assert, t, tc)
		})
	}
}

var noopTransitionFromPaused = NewTestCase(
	&TestInput{
		StartState: StatePaused,
		StateTransitions: StateTransitionMap{
			StatePaused: toStoppedTrFunc,
		},
	},
	&TestExpectation{
		StartState: StatePaused,
		FinalState: StateStopped,
		LoopErr:    nil,
		AllStates: StateList{
			StatePaused,
		},
	},
)

var failedTransitionFromPaused = NewTestCase(
	&TestInput{
		StartState: StatePaused,
		StateTransitions: StateTransitionMap{
			StatePaused: failingTrFunc,
		},
	},
	&TestExpectation{
		StartState: StatePaused,
		FinalState: StateFailed,
		LoopErr:    failingTrFuncError,
		AllStates: StateList{
			StatePaused,
		},
	},
)

var flipflopPausedRunning = NewTestCase(
	&TestInput{
		StartState: StatePaused,
		StateTransitions: StateTransitionMap{
			StatePaused:  toRunningTrFunc,
			StateRunning: toPausedTrFunc,
		},
	},
	&TestExpectation{
		StartState: StatePaused,
		FinalState: StateStopped,
		LoopErr:    nil,
		AllStates: StateList{
			StatePaused,
			StateRunning,
		},
	},
)

var successful10Steps = NewTestCase(
	&TestInput{
		StartState: StatePaused,
		StateTransitions: StateTransitionMap{
			StatePaused:  toRunningTrFunc,
			StateRunning: holdStateForNStepsThenSwitch(StateRunning, 10, StateStopped),
		},
	},
	&TestExpectation{
		StartState: StatePaused,
		FinalState: StateStopped,
		LoopErr:    nil,
		AllStates: StateList{
			StatePaused,
			StateRunning,
		},
	},
)

func startSm(ctx context.Context, assert *require.Assertions, tc *TestCase) (*StateMachine, chan error) {
	sm := NewStateMachine(tc.in.SmId, tc.in.StartState, tc.in.StateTransitions, NoChildren)
	assert.Equal(tc.in.StartState, sm.CurrentState())
	assert.Equal(tc.ex.AllStates, sm.AllStates())

	loopErrChan := make(chan error)
	go func() {
		loopErrChan <- sm.loop(ctx)
	}()

	return sm, loopErrChan
}

func finalState(assert *require.Assertions, sm *StateMachine, loopErr error, tc *TestCase) {
	assert.ErrorIs(loopErr, tc.ex.LoopErr)
	assert.Equal(tc.ex.FinalState, sm.CurrentState())
}

// FIXME: Should be called something like "TestFunctionCollection"
type TestGroup map[string]testCaseFunc

func (tg TestGroup) RunGroup(assert *require.Assertions, t *testing.T, name string, ts TestSet) {
	tcFunc, ok := testGroup[name]
	assert.True(ok)

	t.Run(name, func(t *testing.T) {
		ts.Run(assert, t, tcFunc)
	})
}

var testGroup = TestGroup{
	"withoutCancel": func(assert *require.Assertions, t *testing.T, tc *TestCase) {
		ctx := context.Background()

		sm, loopErrChan := startSm(ctx, assert, tc)

		loopErr := <-loopErrChan
		finalState(assert, sm, loopErr, tc)
	},

	"withCancel": func(assert *require.Assertions, t *testing.T, tc *TestCase) {
		ctx, cancelCtx := context.WithCancel(context.Background())

		sm, loopErrChan := startSm(ctx, assert, tc)

		time.Sleep(100 * time.Millisecond)
		cancelCtx()
		loopErr := <-loopErrChan
		finalState(assert, sm, loopErr, tc)
	},
}

func TestStateMachine_One(t *testing.T) {
	assert := require.New(t)
	t.Parallel()

	var testSetPerGroup = map[string]TestSet{
		// These tests are expected to terminate for internal reasons. This can be either successful or
		// failed, the emphasis is on the fact that the context is not cancelled.
		"withoutCancel": {
			"noopTransitionFromPaused":   noopTransitionFromPaused,
			"failedTransitionFromPaused": failedTransitionFromPaused,
			"successful10Steps":          successful10Steps,
		},

		// These tests are expected to terminate because of the context being cancelled.
		"withCancel": {
			"flipflopPausedRunning": flipflopPausedRunning,
		},
	}

	for testGroupName, testSet := range testSetPerGroup {
		testGroup.RunGroup(assert, t, testGroupName, testSet)
	}
}
