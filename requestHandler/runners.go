package requestHandler

import (
	"angadrive/database"
	"angadrive/runners"
)

// runnerNotifier adapts the websocket pulse functions to the runners.Notifier
// interface. This is what lets the runners package deliver results/errors back
// to users without depending on the websocket layer.
type runnerNotifier struct{}

// NotifyUser sends an arbitrary message to the user identified by token.
func (runnerNotifier) NotifyUser(token string, message map[string]interface{}) {
	genericUserPulse(token, message)
}

// NotifyFileAdded tells the user that a new file was produced by a runner.
func (runnerNotifier) NotifyFileAdded(file database.FileData) {
	UserFilesPulse(FileUpdate{Toggle: true, File: file})
}

// initRunners sets up the runners package with the websocket notifier.
// It must be called once at startup.
func initRunners() {
	runners.Init(runnerNotifier{})
}

// SubmitVideoConversion enqueues a video conversion job on the runners manager.
func SubmitVideoConversion(file database.FileData) error {
	return runners.Submit("video", runners.VideoJob{File: file})
}

// SubmitVideoPreview enqueues a video preview generation job on the runners
// manager. It returns an error if a preview for the same source video is
// already queued or running (the runners manager dedups by source sha256).
func SubmitVideoPreview(file database.FileData) error {
	return runners.Submit("video_preview", runners.VideoPreviewJob{File: file})
}
