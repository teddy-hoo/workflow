package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type WorkFlow struct {
	states map[string]State
	actions map[string]Action
	cancelContext context.Context
	cancelFunc func()
	stateWg sync.WaitGroup
	timeout time.Duration
	startState State
	endStates []State
}

func NewWorkFlow(timeout time.Duration) *WorkFlow {
	wf := WorkFlow{}
	wf.Init(timeout)
	return &wf
}

func (wf *WorkFlow) Init(timeout time.Duration) {
	wf.cancelContext, wf.cancelFunc = context.WithCancel(context.Background())
	wf.stateWg = sync.WaitGroup{}
	wf.timeout = timeout
	wf.states = make(map[string]State)
	wf.actions = make(map[string]Action)
}

func (wf *WorkFlow) StartState() State {
	if wf.startState == nil {
		for _, state := range wf.states {
			if state.PreviousActions() == nil || len(*state.PreviousActions()) == 0 {
				wf.startState = state
			}
		}
	}
	return wf.startState
}

func (wf *WorkFlow) EndState() []State {
	if len(wf.endStates) == 0 {
		for _, state := range wf.states {
			if state.PostActions() == nil || len(*state.PostActions()) == 0 {
				wf.endStates = append(wf.endStates, state)
			}
		}
	}
	return wf.endStates
}

func (wf *WorkFlow) AddEdge(previousState State, action Action, postState State) {
	if _, ok := wf.states[previousState.StateName()]; !ok {
		wf.states[previousState.StateName()] = previousState
		wf.stateWg.Add(1)
	}
	if _, ok := wf.states[postState.StateName()]; !ok {
		wf.states[postState.StateName()] = postState
		wf.stateWg.Add(1)
	}
	wf.actions[action.ActionName()] = action
	previousState.AddPostAction(action)
	previousState.SetWorkFlow(wf)
	postState.AddPreviousAction(action)
	postState.SetWorkFlow(wf)
	action.AddPreviousState(previousState)
	action.AddPostState(postState)
	action.SetWorkFlow(wf)
}

func (wf *WorkFlow) Run() bool {
	if len(wf.states) <= 0 {
		return false
	}
	for name, state := range wf.states {
		s := state
		fmt.Println(name + " entering...")
		go func() {
			s.Enter()
		}()
	}
	ch := make(chan struct{})
	go func() {
		wf.stateWg.Wait()
		close(ch)
	}()
	select {
	case <- wf.cancelContext.Done():
		fmt.Println("Workflow canceled...")
		return false
	case <- ch:
		fmt.Println("Workflow done...")
		return true
	case <- time.After(wf.timeout):
		fmt.Println("Workflow timeout...")
		return false
	}
}

type State interface {
	Enter()
	Leave() <- chan struct{}
	AddPreviousAction(action Action)
	AddPostAction(action Action)
	StateName() string
	SetWorkFlow(workflow *WorkFlow)
	PreviousActions() *[]Action
	PostActions() *[]Action
	PreviousActionDone()
}


type StandardState struct {
	wg sync.WaitGroup
	previousDoneCh chan struct{}
	previousActions []Action
	postActions []Action
	stateName string
	workFlow *WorkFlow
}

func NewState(stateName string) *StandardState {
	state := StandardState{}
	state.stateName = stateName
	return &state
}

func (s *StandardState) PreviousActions() *[]Action {
	return &s.previousActions
}

func (s *StandardState) PreviousActionDone() {
	s.wg.Done()
}

func (s *StandardState) PostActions() *[]Action {
	return &s.postActions
}

func (s *StandardState) SetWorkFlow(workflow *WorkFlow) {
	s.workFlow = workflow
}

func (s *StandardState) StateName() string {
	return s.stateName
}

func (s *StandardState) Enter() {
	fmt.Println(s.stateName + " is waiting...")
	s.wg.Wait()
	fmt.Println(s.stateName + " reached, spawn all post actions...")
	for _, action := range s.postActions {
		a := action
		go func() {
			a.Process()
		}()
	}
	s.Leave()
}

func (s *StandardState) Leave() <- chan struct{} {
	fmt.Println(s.stateName + " leaving...")
	ch := make(chan struct{})
	defer func() {
		close(ch)
	}()
	s.workFlow.stateWg.Done()
	return ch
}

func (s *StandardState) AddPreviousAction(action Action) {
	s.wg.Add(1)
	s.previousActions = append(s.previousActions, action)
}

func (s *StandardState) AddPostAction(action Action) {
	s.postActions = append(s.postActions, action)
}

type Action interface {
	Process()
	AddPreviousState(state State)
	AddPostState(state State)
	ActionName() string
	SetWorkFlow(workflow *WorkFlow)
}


type StandardAction struct {
	previousStates []State
	postStates []State
	timeout time.Duration
	actionName string
	processingFunc func() bool
	doneChan chan struct{}
	failChan chan struct{}
	workFlow *WorkFlow
}

func NewAction(actionName string, timeout time.Duration, processingFunc func() bool) Action {
	action := StandardAction{}
	action.timeout = timeout
	action.actionName = actionName
	action.processingFunc = processingFunc
	action.doneChan = make(chan struct{})
	action.failChan = make(chan struct{})
	return &action
}

func (a *StandardAction) SetWorkFlow(workFlow *WorkFlow) {
	a.workFlow = workFlow
}

func (a *StandardAction) ActionName() string {
	return a.actionName
}

func (a *StandardAction) Process() {
	fmt.Println(a.actionName + " is processing...")
	go func() {
		if a.processingFunc != nil {
			result := a.processingFunc()
			if result {
				close(a.doneChan)
			} else {
				close(a.failChan)
			}
		}
		time.Sleep(time.Second)
	}()
	select {
	case <- a.doneChan:
		for _, s := range a.postStates {
			s.PreviousActionDone()
		}
	case <- a.failChan:
		a.workFlow.cancelFunc()
	case <- time.After(a.timeout):
		a.workFlow.cancelFunc()
	}
}

func (a *StandardAction) AddPreviousState(state State) {
	a.previousStates = append(a.previousStates, state)
}

func (a *StandardAction) AddPostState(state State) {
	a.postStates = append(a.postStates, state)
}
