package main

import (
	"log"
	"sync"

	"go.i3wm.org/i3/v4"
)

type Window struct {
	ID int64
}

type WindowRegistry struct {
	mutex   sync.Mutex
	windows map[int64]*Window
}

func (wr *WindowRegistry) Deregister(id int64) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	if _, ok := wr.windows[id]; ok {
		delete(wr.windows, id)
		log.Printf("Deregistered: %d\n", id)
	}
}

func (wr *WindowRegistry) Register(id int64) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	if _, ok := wr.windows[id]; !ok {
		wr.windows[id] = &Window{ID: id}
		log.Printf("Registered: %d\n", id)
	}
}

func (wr *WindowRegistry) Focus(id int64) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()
}

func main() {
	registry := &WindowRegistry{windows: make(map[int64]*Window, 0)}

	recv := i3.Subscribe(i3.WindowEventType)
	for recv.Next() {
		ev := recv.Event().(*i3.WindowEvent)

		if ev.Container.WindowProperties.Class == "Alacritty" {
			id := ev.Container.Window

			switch ev.Change {
			case "new":
				registry.Register(id)
			case "close":
				registry.Deregister(id)
			case "focus":
				registry.Focus(id)
			}
		}
	}
	log.Fatal(recv.Close())
}
