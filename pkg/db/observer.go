package db

import (
	"sync"
)

// ObserverCode is referring to the type of observer notification.
type ObserverCode int

const (
	ObserveNewLeaderLog       = 0
	ObserveUpdatedBlockStatus = 1
)

// ObserverMessage is a message describing a change of a db.DB update.
type ObserverMessage struct {
	Code     ObserverCode
	Response interface{}
}

// Observer allows registering change listeners for a db.DB instance.
type Observer struct {
	channels []chan<- ObserverMessage
	lock     sync.Mutex
}

// Sub subscribes the given channel to get notification, when the db.DB got
// updated.
func (obv *Observer) Sub(c chan<- ObserverMessage) {
	obv.lock.Lock()
	defer obv.lock.Unlock()
	obv.channels = append(obv.channels, c)
}

// Pub publishes the given message, which is distributed over all subscribed
// channels.
func (obv *Observer) Pub(msg ObserverMessage) {
	obv.lock.Lock()
	defer obv.lock.Unlock()
	for _, c := range obv.channels {
		go push(c, msg)
	}
}

// push is pushing a message to the given channel.
func push(c chan<- ObserverMessage, msg ObserverMessage) {
	c <- msg
}

// Close is closing this observer.
func (obv *Observer) Close() {
	if obv.channels != nil {
		for _, channel := range obv.channels {
			close(channel)
		}
	}
}
