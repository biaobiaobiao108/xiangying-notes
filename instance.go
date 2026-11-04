package main

// windowActivator is the part of a native window needed when a second launch
// asks the first process to come forward.
type windowActivator interface {
	IsMinimized() bool
	Restore()
	Show()
	Focus()
}

type instanceActivation struct {
	window  windowActivator
	pending bool
}

func (a *instanceActivation) activate() {
	if a.window == nil {
		a.pending = true
		return
	}
	a.present()
}

func (a *instanceActivation) setWindow(window windowActivator) {
	a.window = window
	if a.pending {
		a.pending = false
		a.present()
	}
}

func (a *instanceActivation) present() {
	if a.window.IsMinimized() {
		a.window.Restore()
	}
	a.window.Show()
	a.window.Focus()
}
