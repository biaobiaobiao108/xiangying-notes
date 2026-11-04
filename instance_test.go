package main

import "testing"

type fakeActivationWindow struct {
	minimized bool
	shown     int
	focused   int
	restored  int
}

func (w *fakeActivationWindow) IsMinimized() bool { return w.minimized }
func (w *fakeActivationWindow) Restore()          { w.restored++; w.minimized = false }
func (w *fakeActivationWindow) Show()             { w.shown++ }
func (w *fakeActivationWindow) Focus()            { w.focused++ }

func TestSecondLaunchRestoresAndFocusesWindow(t *testing.T) {
	window := &fakeActivationWindow{minimized: true}
	activation := &instanceActivation{window: window}
	activation.activate()
	if window.restored != 1 || window.shown != 1 || window.focused != 1 {
		t.Fatalf("activation actions = restore:%d show:%d focus:%d", window.restored, window.shown, window.focused)
	}
}

func TestSecondLaunchDuringStartupActivatesWindowWhenReady(t *testing.T) {
	activation := &instanceActivation{}
	activation.activate()
	if !activation.pending {
		t.Fatal("activation should wait for the main window")
	}
	window := &fakeActivationWindow{}
	activation.setWindow(window)
	if activation.pending || window.shown != 1 || window.focused != 1 {
		t.Fatalf("pending=%v show=%d focus=%d", activation.pending, window.shown, window.focused)
	}
}
