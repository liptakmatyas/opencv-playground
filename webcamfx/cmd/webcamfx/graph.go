package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var TeardownTimedOut = errors.New("teardown timed out")

type namedNodes map[string]Node

type Graph struct {
	name               string
	isRunningMu        sync.Mutex
	nodes              namedNodes
	errChan            chan error
	minTeardownTimeout time.Duration
}

// The Graph is a Node.
//
// This allows for nesting Graphs inside Graphs.
var _ Node = &Graph{}

func NewGraph(name string) *Graph {
	return &Graph{
		name:               name,
		isRunningMu:        sync.Mutex{},
		nodes:              make(namedNodes, 0),
		errChan:            make(chan error),
		minTeardownTimeout: 1 * time.Second,
	}
}

func (g *Graph) Name() string {
	return g.name
}

func (g *Graph) SetNode(node Node) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	g.nodes[node.Name()] = node
}

func (g *Graph) SetNodes(nodes ...Node) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	for _, node := range nodes {
		g.nodes[node.Name()] = node
	}
}

func (g *Graph) RemoveNode(node Node) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	delete(g.nodes, node.Name())
}

func (g *Graph) RemoveNodes(nodes ...Node) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	for _, node := range nodes {
		delete(g.nodes, node.Name())
	}
}

func (g *Graph) RemoveNodeByName(name string) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	delete(g.nodes, name)
}

func (g *Graph) RemoveNodesByName(names ...string) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	for _, name := range names {
		delete(g.nodes, name)
	}
}

func (g *Graph) Run(ctx context.Context) {
	go g.loop(ctx)
}

func (g *Graph) Err() <-chan error {
	return g.errChan
}

func (g *Graph) sendErr(err error) {
	g.errChan <- err
}

func (g *Graph) teardownTimeoutForNNodes(n int) time.Duration {
	if n <= 0 {
		return g.minTeardownTimeout
	}

	return g.minTeardownTimeout + time.Duration(math.Log10(float64(n)))
}

func (g *Graph) loop(parentCtx context.Context) {
	g.isRunningMu.Lock()
	defer g.isRunningMu.Unlock()

	var contextCanceledIsAnError = false
	var runningNodes = make(map[string]Node, 0)
	var nodeErrs = make(map[string]error, 0)

	nodeCtx, cancelNodeCtx := context.WithCancel(parentCtx)
	for nodeName, node := range g.nodes {
		node.Run(nodeCtx)
		runningNodes[nodeName] = node
	}

	mainLoopErr := func(ctx context.Context) error {
		for {
			for nodeName, node := range runningNodes {
				select {
				case <-ctx.Done():
					if contextCanceledIsAnError {
						return context.Canceled
					}
					return nil

				case err := <-node.Err():
					logger.
						WithField("node", nodeName).
						WithError(err).
						Errorf("Node error")

					nodeErrs[nodeName] = err
					delete(runningNodes, nodeName)
					return errors.New("Node error")

				default:
					continue
				}
			}
		}
	}(parentCtx)

	teardownErr := func() error {
		if len(runningNodes) == 0 {
			return nil
		}

		teardownTimeout := time.NewTimer(g.teardownTimeoutForNNodes(len(runningNodes)))
		cancelNodeCtx()

	TEARDOWN_LOOP:
		for {
			if len(runningNodes) == 0 {
				return nil
			}

			for nodeName, node := range runningNodes {
				select {
				case <-teardownTimeout.C:
					logger.Warn("Teardown timeout.")
					return TeardownTimedOut

				case err := <-node.Err():
					if err != nil {
						if !errors.Is(err, context.Canceled) || contextCanceledIsAnError {
							logger.Tracef("Node '%s' error: %v", nodeName, err)
							nodeErrs[nodeName] = err
						}
					}
					delete(runningNodes, nodeName)
					continue TEARDOWN_LOOP

				default:
					continue
				}
			}
		}
	}()

	var errMsg string
	if mainLoopErr != nil {
		errMsg += fmt.Sprintf("[MainLoop: %s]", mainLoopErr.Error())
	}
	if teardownErr != nil {
		errMsg += fmt.Sprintf("[TearDown: %s]", teardownErr.Error())
	}
	for nodeName, err := range nodeErrs {
		if err != nil {
			errMsg += fmt.Sprintf("[Node %s: %v]", nodeName, err)
		}
	}

	var err error = nil
	if errMsg != "" {
		err = errors.Errorf("Graph %s failed: %s", g.name, errMsg)
	}
	g.sendErr(err)
	return
}
