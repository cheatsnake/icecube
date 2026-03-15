package jobs

import "testing"

func TestJobStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     JobStatus
		to       JobStatus
		expected bool
	}{
		// From Pending
		{"pending -> processing", JobStatusPending, JobStatusProcessing, true},
		{"pending -> failed", JobStatusPending, JobStatusFailed, true},
		{"pending -> completed", JobStatusPending, JobStatusCompleted, false},
		{"pending -> pending", JobStatusPending, JobStatusPending, false},

		// From Processing
		{"processing -> completed", JobStatusProcessing, JobStatusCompleted, true},
		{"processing -> failed", JobStatusProcessing, JobStatusFailed, true},
		{"processing -> pending", JobStatusProcessing, JobStatusPending, false},
		{"processing -> processing", JobStatusProcessing, JobStatusProcessing, false},

		// From Completed
		{"completed -> anything", JobStatusCompleted, JobStatusProcessing, false},
		{"completed -> failed", JobStatusCompleted, JobStatusFailed, false},

		// From Failed
		{"failed -> anything", JobStatusFailed, JobStatusProcessing, false},
		{"failed -> pending", JobStatusFailed, JobStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo(%q -> %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}
