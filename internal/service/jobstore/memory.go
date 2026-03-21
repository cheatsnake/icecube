package jobstore

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/jobs"
)

type JobStoreMemory struct {
	logger      *slog.Logger
	mu          sync.RWMutex
	jobs        map[string]*jobs.Job
	tasks       map[string]*jobs.Task
	notifyCh    chan struct{}
	subscribers []chan struct{}
}

func NewJobStoreMemory(logger *slog.Logger) *JobStoreMemory {
	return &JobStoreMemory{
		logger:      logger,
		jobs:        make(map[string]*jobs.Job),
		tasks:       make(map[string]*jobs.Task),
		notifyCh:    make(chan struct{}, 1),
		subscribers: make([]chan struct{}, 0),
	}
}

func (s *JobStoreMemory) CreateJob(ctx context.Context, job *jobs.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		s.logger.Error("Job already exists", "jobID", job.ID)
		return errors.Join(errs.ErrAlreadyExists, errors.New("job already exists: "+job.ID))
	}

	jobCopy := *job
	jobCopy.Tasks = make([]*jobs.Task, len(job.Tasks))

	for i, task := range job.Tasks {
		taskCopy := *task
		jobCopy.Tasks[i] = &taskCopy
		s.tasks[task.ID] = &taskCopy
	}

	s.jobs[job.ID] = &jobCopy
	s.logger.Debug("Job created", "jobID", job.ID, "taskCount", len(job.Tasks))
	s.notifySubscribers()
	return nil
}

func (s *JobStoreMemory) SubscribeOnJob() chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan struct{}, 1)
	s.subscribers = append(s.subscribers, ch)
	return ch
}

func (s *JobStoreMemory) UnsubscribeOnJob(ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.subscribers {
		if sub == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

func (s *JobStoreMemory) GetJob(ctx context.Context, id string) (*jobs.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		s.logger.Debug("Job not found", "jobID", id)
		return nil, errors.Join(errs.ErrNotFound, errors.New("job not found: "+id))
	}

	jobCopy := *job
	jobCopy.Tasks = make([]*jobs.Task, len(job.Tasks))

	taskIndex := 0
	for _, task := range s.tasks {
		if task.JobID == id {
			taskCopy := *task
			jobCopy.Tasks[taskIndex] = &taskCopy
			taskIndex++
		}
	}

	jobCopy.Tasks = jobCopy.Tasks[:taskIndex]

	s.logger.Debug("Job retrieved", "jobID", id)
	return &jobCopy, nil
}

func (s *JobStoreMemory) AcquireJob(ctx context.Context) (*jobs.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pendingJobs []*jobs.Job
	for _, job := range s.jobs {
		if job.Status == jobs.JobStatusPending {
			pendingJobs = append(pendingJobs, job)
		}
	}

	if len(pendingJobs) == 0 {
		s.logger.Debug("No pending jobs available")
		return nil, nil
	}

	sort.Slice(pendingJobs, func(i, j int) bool {
		return pendingJobs[i].CreatedAt.Before(pendingJobs[j].CreatedAt)
	})

	job := pendingJobs[0]

	job.Status = jobs.JobStatusProcessing
	now := time.Now().UTC()
	job.LockedAt = &now

	jobCopy := *job
	jobCopy.Tasks = make([]*jobs.Task, 0)

	for _, task := range s.tasks {
		if task.JobID == job.ID {
			taskCopy := *task
			jobCopy.Tasks = append(jobCopy.Tasks, &taskCopy)
		}
	}

	s.logger.Debug("Job acquired", "jobID", job.ID, "originalID", job.OriginalID)
	return &jobCopy, nil
}

func (s *JobStoreMemory) ReleaseJobs(ctx context.Context, lease time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	cutoff := now.Add(-lease)

	for _, job := range s.jobs {
		if job.Status == jobs.JobStatusProcessing && job.LockedAt != nil {
			if job.LockedAt.Before(cutoff) {
				job.Status = jobs.JobStatusPending
				job.LockedAt = nil
			}
		}
	}

	return nil
}

func (s *JobStoreMemory) UpdateJob(ctx context.Context, job *jobs.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingJob, exists := s.jobs[job.ID]
	if !exists {
		s.logger.Error("Job not found for update", "jobID", job.ID)
		return errors.Join(errs.ErrNotFound, errors.New("job not found: "+job.ID))
	}

	existingJob.Status = job.Status
	existingJob.LockedAt = job.LockedAt
	existingJob.Reason = job.Reason

	s.logger.Debug("Job updated", "jobID", job.ID, "status", job.Status)
	return nil
}

func (s *JobStoreMemory) DeleteJob(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; !exists {
		s.logger.Error("Job not found for deletion", "jobID", id)
		return errors.Join(errs.ErrNotFound, errors.New("job not found: "+id))
	}

	delete(s.jobs, id)

	for taskID, task := range s.tasks {
		if task.JobID == id {
			delete(s.tasks, taskID)
		}
	}

	s.logger.Debug("Job deleted", "jobID", id)
	return nil
}

func (s *JobStoreMemory) UpdateTask(ctx context.Context, task *jobs.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingTask, exists := s.tasks[task.ID]
	if !exists {
		s.logger.Error("Task not found for update", "taskID", task.ID)
		return errors.Join(errs.ErrNotFound, errors.New("task not found: "+task.ID))
	}

	existingTask.VariantID = task.VariantID

	return nil
}

func (s *JobStoreMemory) UpdateTasks(ctx context.Context, tasks []*jobs.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range tasks {
		if _, exists := s.tasks[task.ID]; !exists {
			s.logger.Error("Task not found for update", "taskID", task.ID)
			return errors.Join(errs.ErrNotFound, errors.New("task not found: "+task.ID))
		}
	}

	for _, task := range tasks {
		s.tasks[task.ID].VariantID = task.VariantID
	}

	s.logger.Debug("Tasks updated", "count", len(tasks))
	return nil
}

func (s *JobStoreMemory) notifySubscribers() {
	s.logger.Debug("Notifying subscribers", "count", len(s.subscribers))
	for _, ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
