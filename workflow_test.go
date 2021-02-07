package workflow

import (
	"testing"
	"time"
)

func processTestFunc() bool {
	time.Sleep(time.Second * 2)
	return true
}

func processTestFailFunc() bool {
	time.Sleep(time.Second * 2)
	return false
}

func processTestTimeoutFunc() bool {
	time.Sleep(time.Second * 6)
	return false
}

func TestWorkFlow1 (t *testing.T) {
	wf := NewWorkFlow(time.Second * 5)
	s1 := NewState("start")
	s2 := NewState("end")
	a1 := NewAction("action1", time.Second * 3, processTestFunc)
	wf.AddEdge(s1, a1, s2)
	if wf.Run() != true {
		t.Fail()
	}
}

func TestWorkFlow2 (t *testing.T) {
	wf := NewWorkFlow(time.Second * 5)
	s1 := NewState("start")
	s2 := NewState("end")
	a1 := NewAction("action1", time.Second * 3, processTestFailFunc)
	wf.AddEdge(s1, a1, s2)
	if wf.Run() != false {
		t.Fail()
	}
}

func TestWorkFlow3 (t *testing.T) {
	wf := NewWorkFlow(time.Second * 5)
	s1 := NewState("start")
	s2 := NewState("end")
	a1 := NewAction("action1", time.Second * 3, processTestTimeoutFunc)
	wf.AddEdge(s1, a1, s2)
	if wf.Run() != false {
		t.Fail()
	}
}
