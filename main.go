package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"go.i3wm.org/i3/v4"
)

type CustomEvent interface {
}

type EWindowFocus struct {
	Window
}

type EWindowBlur struct {
	Window
}

type Window struct {
	ID    int64
	Class string
}

type WindowRegistry struct {
	mutex       sync.Mutex
	windows     map[int64]*Window
	lastFocused int64
}

func (wr *WindowRegistry) Deregister(id int64) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	if _, ok := wr.windows[id]; ok {
		Events <- EWindowBlur{Window: *wr.windows[id]}
		wr.lastFocused = -1

		delete(wr.windows, id)
		// log.Printf("Deregistered: %d\n", id)
	}
}

func (wr *WindowRegistry) Register(id int64, class string) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	if _, ok := wr.windows[id]; !ok {
		wr.windows[id] = &Window{ID: id, Class: class}
		// log.Printf("Registered: %d\n", id)
	}
}

func (wr *WindowRegistry) Focus(id int64) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	old := wr.lastFocused
	wr.lastFocused = id

	if old != -1 && wr.exists(old) {
		Events <- EWindowBlur{Window: *wr.windows[old]}
	}
	if old != id && wr.exists(id) {
		Events <- EWindowFocus{Window: *wr.windows[id]}
	}
}

func (wr *WindowRegistry) exists(id int64) bool {
	_, ok := wr.windows[id]
	return ok
}

var (
	Registry *WindowRegistry
	Events   chan (CustomEvent)
)

func onOutputEvent(ev *i3.OutputEvent) {
	log.Println(ev)
}

func onWindowEvent(ev *i3.WindowEvent) {
	// if ev.Container.WindowProperties.Class == "Alacritty" {
	id := ev.Container.Window

	switch ev.Change {
	case "new":
		Registry.Register(id, ev.Container.WindowProperties.Class)
	case "close":
		Registry.Deregister(id)
	case "focus":
		Registry.Focus(id)
	}
	// }
}

func customEventLoop() {
	for ev := range Events {
		switch ev.(type) {
		case EWindowBlur:
			blur := ev.(EWindowBlur)
			log.Printf("Blur: %d/%s\n", blur.Window.ID, blur.Window.Class)
		case EWindowFocus:
			focus := ev.(EWindowFocus)
			log.Printf("Focus: %d/%s\n", focus.Window.ID, focus.Window.Class)
		}
	}
}

func init() {
	Registry = &WindowRegistry{
		windows:     make(map[int64]*Window, 0),
		lastFocused: -1,
	}
	Events = make(chan CustomEvent)
}

func main() {
	version, err := i3.GetVersion()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Println("i3:", version.HumanReadable)

	windowCh := make(chan *i3.WindowEvent, 10)
	outputCh := make(chan *i3.OutputEvent, 10)

	go customEventLoop()

	go func() {
		for ev := range windowCh {
			onWindowEvent(ev)
		}
	}()

	go func() {
		for ev := range outputCh {
			onOutputEvent(ev)
		}
	}()

	recv := i3.Subscribe(
		i3.WindowEventType,
		i3.OutputEventType,
	)

	for recv.Next() {
		ev := recv.Event()
		switch ev.(type) {
		case *i3.WindowEvent:
			windowCh <- ev.(*i3.WindowEvent)
		case *i3.OutputEvent:
			outputCh <- ev.(*i3.OutputEvent)
		}

	}

	close(windowCh)
	close(outputCh)
	close(Events)

	log.Fatal(recv.Close())
}
