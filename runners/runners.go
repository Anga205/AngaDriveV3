// Package runners provides a generic framework for queueing and running
// compute-heavy background tasks ("runners").
//
// Each compute-heavy feature (video conversion, and future features such as
// image processing, document rendering, etc.) is implemented as a Runner that
// executes Jobs. A single Manager queues jobs and runs them with bounded
// concurrency, so only a few CPU-intensive tasks run at any given time.
//
// The package is deliberately decoupled from the websocket layer: results and
// errors are delivered back to the user through a Notifier interface that the
// caller implements. This keeps the runners package self-contained and makes
// it easy to later replace the local execution with a remote API call.
package runners

import (
	"angadrive/database"
	"fmt"
	"os"
	"strconv"
	"sync"
)

// Job is a unit of work that can be queued and executed by a Runner.
//
// ID() must return a stable, unique key for the job. It is used for
// deduplication: a job with the same ID cannot be queued or running twice at
// the same time.
type Job interface {
	ID() string
}

// Runner executes a Job. Each compute-heavy feature is implemented as a Runner.
type Runner interface {
	// Name is the unique identifier used to submit jobs to this runner.
	Name() string
	// Run performs the work for a single job. It is called from a worker
	// goroutine and must be safe to run concurrently with other jobs.
	Run(job Job) error
}

// Notifier delivers results and errors back to the user (e.g. over websocket).
//
// It is implemented by the caller so the runners package does not depend on
// the websocket layer. This is what makes it possible to swap the local
// execution for a remote API call later: the runner only talks to the
// Notifier, never to the transport directly.
type Notifier interface {
	// NotifyUser sends an arbitrary message to the user identified by token.
	NotifyUser(token string, message map[string]interface{})
	// NotifyFileAdded tells the user that a new file was produced by a runner.
	NotifyFileAdded(file database.FileData)
}

// queuedJob pairs a job with the name of the runner that should execute it.
type queuedJob struct {
	runnerName string
	job        Job
}

// Manager queues jobs and runs them with bounded concurrency.
//
// It owns a buffered queue (the backlog), a semaphore (the concurrency limit),
// and a dedup map (so the same job is never queued or running twice).
type Manager struct {
	queue     chan queuedJob
	semaphore chan struct{}
	inFlight  sync.Map // job.ID() -> true
	runners   map[string]Runner
	notifier  Notifier
}

// NewManager creates a Manager with the given concurrency limit and queue size.
//
//	maxConcurrent: maximum number of jobs running at the same time.
//	queueSize:     maximum number of jobs waiting in the backlog.
//	notifier:      delivers results/errors back to the user.
func NewManager(maxConcurrent, queueSize int, notifier Notifier) *Manager {
	m := &Manager{
		queue:     make(chan queuedJob, queueSize),
		semaphore: make(chan struct{}, maxConcurrent),
		runners:   make(map[string]Runner),
		notifier:  notifier,
	}
	go m.worker()
	return m
}

// Register adds a Runner to the manager so jobs can be submitted to it by name.
func (m *Manager) Register(r Runner) {
	m.runners[r.Name()] = r
}

// Submit enqueues a job for the named runner. It is non-blocking: if the queue
// is full, or the job is already queued/running, it returns an error.
func (m *Manager) Submit(runnerName string, job Job) error {
	// Atomically mark the job as in-flight; reject duplicates.
	if _, loaded := m.inFlight.LoadOrStore(job.ID(), true); loaded {
		return fmt.Errorf("job %s already queued or running", job.ID())
	}
	if _, ok := m.runners[runnerName]; !ok {
		m.inFlight.Delete(job.ID())
		return fmt.Errorf("unknown runner %q", runnerName)
	}
	// Non-blocking enqueue; roll back the marker if the queue is full.
	select {
	case m.queue <- queuedJob{runnerName: runnerName, job: job}:
		return nil
	default:
		m.inFlight.Delete(job.ID())
		return fmt.Errorf("runner queue is full")
	}
}

// worker drains the queue and fans each job out to its own goroutine, bounded
// by the semaphore. The worker goroutine itself never blocks on the job's work;
// it only blocks briefly on the semaphore before handing the job off.
func (m *Manager) worker() {
	for qj := range m.queue {
		// Acquire a concurrency slot (block if the limit is reached).
		m.semaphore <- struct{}{}
		go func(qj queuedJob) {
			// Release the slot and clear the dedup marker when done.
			defer func() { <-m.semaphore }()
			defer m.inFlight.Delete(qj.job.ID())
			r := m.runners[qj.runnerName]
			if r == nil {
				return
			}
			r.Run(qj.job)
		}(qj)
	}
}

// Default limits for the package-level manager.
const (
	DefaultMaxConcurrent = 5
	DefaultQueueSize     = 100
)

// envInt reads an integer from an environment variable, falling back to
// defaultValue if the variable is unset or not a valid integer.
func envInt(name string, defaultValue int) int {
	if raw := os.Getenv(name); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			return v
		}
	}
	return defaultValue
}

var defaultManager *Manager

// Init sets up the package-level default manager and registers the built-in
// runners. It must be called once at startup, before any Submit calls.
//
// The concurrency limit and queue size are read from environment variables:
//
//	MAX_RUNNER_COUNT: maximum number of jobs running at the same time
//	                  (default 5).
//	MAX_QUEUE_SIZE:   maximum number of jobs waiting in the backlog
//	                  (default 100).
func Init(notifier Notifier) {
	maxConcurrent := envInt("MAX_RUNNER_COUNT", DefaultMaxConcurrent)
	queueSize := envInt("MAX_QUEUE_SIZE", DefaultQueueSize)
	defaultManager = NewManager(maxConcurrent, queueSize, notifier)
	defaultManager.Register(&VideoRunner{notifier: notifier})
	defaultManager.Register(&VideoPreviewRunner{notifier: notifier})
	// Start the backfill loop so any videos without previews get one generated
	// whenever the manager is idle.
	StartBackfillLoop(defaultManager)
}

// Submit enqueues a job on the default manager.
func Submit(runnerName string, job Job) error {
	if defaultManager == nil {
		return fmt.Errorf("runners not initialized")
	}
	return defaultManager.Submit(runnerName, job)
}
