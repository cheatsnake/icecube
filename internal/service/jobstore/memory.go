package jobstore

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cheatsnake/icm/internal/domain/jobs"
)

type JobStoreMemory struct {
	mu    sync.RWMutex
	jobs  map[string]*jobs.Job
	tasks map[string]*jobs.Task
}

func NewJobStoreMemory() (*JobStoreMemory, error) {
	return &JobStoreMemory{
		jobs:  make(map[string]*jobs.Job),
		tasks: make(map[string]*jobs.Task),
	}, nil
}

func (s *JobStoreMemory) CreateJob(ctx context.Context, job *jobs.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job already exists: %s", job.ID)
	}

	jobCopy := *job
	jobCopy.Tasks = make([]*jobs.Task, len(job.Tasks))

	for i, task := range job.Tasks {
		taskCopy := *task
		jobCopy.Tasks[i] = &taskCopy
		s.tasks[task.ID] = &taskCopy
	}

	s.jobs[job.ID] = &jobCopy
	return nil
}

func (s *JobStoreMemory) GetJob(ctx context.Context, id string) (*jobs.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", id)
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
		return fmt.Errorf("job not found: %s", job.ID)
	}

	existingJob.Status = job.Status
	existingJob.LockedAt = job.LockedAt

	return nil
}

func (s *JobStoreMemory) DeleteJob(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	delete(s.jobs, id)

	for taskID, task := range s.tasks {
		if task.JobID == id {
			delete(s.tasks, taskID)
		}
	}

	return nil
}

func (s *JobStoreMemory) UpdateTask(ctx context.Context, task *jobs.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingTask, exists := s.tasks[task.ID]
	if !exists {
		return fmt.Errorf("task not found: %s", task.ID)
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
			return fmt.Errorf("task not found: %s", task.ID)
		}
	}

	for _, task := range tasks {
		s.tasks[task.ID].VariantID = task.VariantID
	}

	return nil
}
