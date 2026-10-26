package requestHandler

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/protocol"
	"angadrive/runners"
)

// runnerNotifier adapts the hub broadcast functions to the runners.Notifier
// interface, keeping the runners package independent of the transport layer.
type runnerNotifier struct{}

func (runnerNotifier) NotifyUser(token string, message map[string]interface{}) {
	hub.NotifyUser(token, message)
}

func (runnerNotifier) NotifyFileAdded(file database.FileData) {
	hub.PublishFileUpdate(protocol.FileUpdate{Toggle: true, File: file})
}

var _ runners.Notifier = runnerNotifier{}
