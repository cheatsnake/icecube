package processor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/jobs"
	"github.com/cheatsnake/icecube/internal/domain/processing"
	"github.com/cheatsnake/icecube/internal/service/imagestore"
	"github.com/cheatsnake/icecube/internal/service/jobstore"
)

type mockProcessor struct {
	processedPath    string
	processErr       error
	processCallCount int
}

func (m *mockProcessor) Process(imagePath string, options *processing.Options) (string, error) {
	m.processCallCount++
	if m.processErr != nil {
		return "", m.processErr
	}
	return m.processedPath, nil
}

type mockJobStore struct {
	acquireJob     *jobs.Job
	acquireErr     error
	updateJobErr   error
	updateTasksErr error
	jobs           map[string]*jobs.Job
	tasks          map[string]*jobs.Task
}

func newMockJobStore() *mockJobStore {
	return &mockJobStore{
		jobs:  make(map[string]*jobs.Job),
		tasks: make(map[string]*jobs.Task),
	}
}

func (m *mockJobStore) AcquireJob(ctx context.Context) (*jobs.Job, error) {
	if m.acquireErr != nil {
		return nil, m.acquireErr
	}
	if m.acquireJob == nil {
		return nil, nil
	}
	// Return a copy to avoid mutation issues
	jobCopy := *m.acquireJob
	return &jobCopy, nil
}

func (m *mockJobStore) ReleaseJobs(ctx context.Context, lease time.Duration) error {
	return nil
}

func (m *mockJobStore) UpdateJob(ctx context.Context, job *jobs.Job) error {
	if m.updateJobErr != nil {
		return m.updateJobErr
	}
	if m.jobs != nil {
		m.jobs[job.ID] = job
	}
	return nil
}

func (m *mockJobStore) UpdateTasks(ctx context.Context, tasks []*jobs.Task) error {
	if m.updateTasksErr != nil {
		return m.updateTasksErr
	}
	if m.tasks != nil {
		for _, t := range tasks {
			m.tasks[t.ID] = t
		}
	}
	return nil
}

func (m *mockJobStore) CountPendingJobs(ctx context.Context) (int, error) {
	return 0, nil
}

type mockImageStore struct {
	metadata      *image.Variant
	metadataErr   error
	downloadData  []byte
	downloadErr   error
	uploadedData  []byte
	uploadVariant *image.Variant
	uploadErr     error
}

func newMockImageStore() *mockImageStore {
	return &mockImageStore{}
}

func (m *mockImageStore) UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	data, _ := io.ReadAll(r)
	m.uploadedData = data
	if m.uploadVariant != nil {
		return m.uploadVariant, nil
	}
	return &image.Variant{
		ID:           "test-variant-id",
		OriginalName: name,
		Format:       image.FormatJPEG,
		Width:        100,
		Height:       100,
		ByteSize:     size,
	}, nil
}

func (m *mockImageStore) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	return io.NopCloser(bytes.NewReader(m.downloadData)), nil
}

func (m *mockImageStore) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	if m.metadataErr != nil {
		return nil, m.metadataErr
	}
	if m.metadata == nil {
		return &image.Variant{
			ID:           id,
			OriginalName: "test",
			Format:       image.FormatJPEG,
			Width:        800,
			Height:       600,
			ByteSize:     1000,
		}, nil
	}
	return m.metadata, nil
}

func createTestJob() *jobs.Job {
	job, _ := jobs.NewJob("original-image-id")
	opt, _ := processing.NewOptions(image.FormatWEBP, 800, 80, false, nil)
	job.AddTask(opt)
	return job
}

func TestWorker_Run_NoJob(t *testing.T) {
	// Arrange
	proc := &mockProcessor{}
	js := newMockJobStore()
	js.acquireJob = nil // No job available
	is := newMockImageStore()
	logger := slog.Default()

	worker := NewWorker(proc, js, is, nil, logger)
	err := worker.Run()

	if err != nil {
		t.Errorf("expected no error when no job, got %v", err)
	}
}

func TestWorker_Run_WithJob(t *testing.T) {
	job := createTestJob()
	proc := &mockProcessor{
		processedPath: "/tmp/processed_test.jpg",
	}
	js := newMockJobStore()
	js.acquireJob = job
	is := newMockImageStore()
	is.downloadData = []byte("fake image data")
	logger := slog.Default()
	worker := NewWorker(proc, js, is, nil, logger)
	_ = worker
}

func TestWorker_Run_AcquireError(t *testing.T) {
	proc := &mockProcessor{}
	js := newMockJobStore()
	js.acquireErr = errors.New("database error")
	is := newMockImageStore()
	logger := slog.Default()
	worker := NewWorker(proc, js, is, nil, logger)

	err := worker.Run()
	if err == nil {
		t.Error("expected error when acquire fails")
	}
}

func TestWorker_Run_DownloadError(t *testing.T) {
	job := createTestJob()
	proc := &mockProcessor{}
	js := newMockJobStore()
	js.acquireJob = job
	is := newMockImageStore()
	is.downloadErr = errors.Join(errs.ErrNotFound, errors.New("image not found"))
	logger := slog.Default()
	worker := NewWorker(proc, js, is, nil, logger)

	err := worker.Run()
	if err == nil {
		t.Error("expected error when download fails")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"not found error", errs.ErrNotFound, false},
		{"invalid input error", errs.ErrInvalidInput, false},
		{"conversion not supported", processing.ErrConversionNotSupported, false},
		{"random error", errors.New("some error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestWorker_WithRealInMemoryStores(t *testing.T) {
	blobStore := imagestore.NewTestBlobStoreMemory(slog.Default())
	metadataStore := imagestore.NewTestMetadataStoreMemory(slog.Default())
	imageStore := imagestore.NewStore(blobStore, metadataStore, slog.Default())

	testImageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // Minimal JPEG header
	metadata, err := imageStore.UploadImage(context.Background(), bytes.NewReader(testImageData), "test.jpg", int64(len(testImageData)))
	if err != nil {
		t.Skipf("skipping test: cannot upload test image: %v", err)
	}

	job, _ := jobs.NewJob(metadata.ID)
	opt, _ := processing.NewOptions(image.FormatJPEG, 100, 80, false, nil)
	job.AddTask(opt)

	js, _ := jobstore.New(jobstore.Config{Type: "memory"}, nil, slog.Default())
	err = js.CreateJob(context.Background(), job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	proc := &mockProcessor{
		processedPath: "/tmp/nonexistent.jpg",
	}

	logger := slog.Default()
	worker := NewWorker(proc, js, imageStore, nil, logger)

	err = worker.Run()
	if err == nil {
		t.Error("expected error due to non-existent processed file")
	}
}
