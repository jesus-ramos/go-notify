// Package notify enables independent components of an application to
// observe notable events in a decoupled fashion.
//
// It generalizes the pattern of *multiple* consumers of an event (ie:
// the same message delivered to multiple channels) and obviates the need
// for components to have intimate knowledge of each other (only `import notify`
// and the name of the event are shared).
//
// Example:
//     notifier := notify.NewNotifier()
//     // producer of "my_event"
//     go func() {
//         for {
//             time.Sleep(time.Duration(1) * time.Second):
//             notifier.Post("my_event", time.Now().Unix())
//         }
//     }()
//
//     // observer of "my_event" (normally some independent component that
//     // needs to be notified when "my_event" occurs)
//     myEventChan := make(chan interface{})
//     notifier.Start("my_event", myEventChan)
//     go func() {
//         for {
//             data := <-myEventChan
//             log.Printf("MY_EVENT: %#v", data)
//         }
//     }()
package notify

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrEventNotFound = errors.New("Event not found")
	ErrPostTimedOut  = errors.New("Post event timed out")
)

// returns the current version
func Version() string {
	return "0.3"
}

type Notifier struct {
	events map[string][]chan interface{}
	sync.RWMutex
}

func NewNotifier() *Notifier {
	return &Notifier{
		events: make(map[string][]chan interface{}),
	}
}

// Start observing the specified event via provided output channel
func (notifier *Notifier) Start(event string, outputChan chan interface{}) {
	notifier.Lock()
	defer notifier.Unlock()

	notifier.events[event] = append(notifier.events[event], outputChan)
}

// Stop observing the specified event on the provided output channel
func (notifier *Notifier) Stop(event string, outputChan chan interface{}) error {
	notifier.Lock()
	defer notifier.Unlock()

	newArray := make([]chan interface{}, 0)
	outChans, ok := notifier.events[event]
	if !ok {
		return ErrEventNotFound
	}
	for _, ch := range outChans {
		if ch != outputChan {
			newArray = append(newArray, ch)
		} else {
			close(ch)
		}
	}
	notifier.events[event] = newArray

	return nil
}

// Stop observing the specified event on all channels
func (notifier *Notifier) StopAll(event string) error {
	notifier.Lock()
	defer notifier.Unlock()

	outChans, ok := notifier.events[event]
	if !ok {
		return ErrEventNotFound
	}
	for _, ch := range outChans {
		close(ch)
	}
	delete(notifier.events, event)

	return nil
}

// Post a notification (arbitrary data) to the specified event
func (notifier *Notifier) Post(event string, data interface{}) error {
	notifier.RLock()
	defer notifier.RUnlock()

	outChans, ok := notifier.events[event]
	if !ok {
		return ErrEventNotFound
	}
	for _, outputChan := range outChans {
		outputChan <- data
	}

	return nil
}

// Post a notification to the specified event using the provided timeout for
// any output channels that are blocking
func (notifier *Notifier) PostTimeout(event string, data interface{}, timeout time.Duration) error {
	notifier.RLock()
	defer notifier.RUnlock()

	var err error = nil

	outChans, ok := notifier.events[event]
	if !ok {
		return ErrEventNotFound
	}
	for _, outputChan := range outChans {
		select {
		case outputChan <- data:
		case <-time.After(timeout):
			err = ErrPostTimedOut
		}
	}

	return err
}
