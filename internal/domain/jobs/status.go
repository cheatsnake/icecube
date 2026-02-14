package jobs

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

func (s JobStatus) CanTransitionTo(next JobStatus) bool {
	switch s {
	case JobStatusPending:
		return next == JobStatusProcessing || next == JobStatusFailed
	case JobStatusProcessing:
		return next == JobStatusCompleted || next == JobStatusFailed
	case JobStatusCompleted, JobStatusFailed:
		return false
	default:
		return false
	}
}
