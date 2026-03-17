package processor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/processing"
)

type mockResizer struct {
	resizeErr error
}

func (m *mockResizer) Resize(params ResizerParams) error {
	return m.resizeErr
}

type mockConverter struct {
	convertErr error
}

func (m *mockConverter) Convert(imagePath, resultPath string) error {
	return m.convertErr
}

type mockCompressor struct {
	compressErr error
}

func (m *mockCompressor) Compress(params CompressorParams) error {
	return m.compressErr
}

type serviceForTest struct {
	resizer    *mockResizer
	converter  *mockConverter
	compressor *mockCompressor
	*Service
}

func newServiceForTest() (*serviceForTest, error) {
	resizer := &mockResizer{}
	converter := &mockConverter{}
	compressor := &mockCompressor{}

	svc := &Service{
		resizer:    resizer,
		converter:  converter,
		compressor: compressor,
	}

	return &serviceForTest{
		resizer:    resizer,
		converter:  converter,
		compressor: compressor,
		Service:    svc,
	}, nil
}

func TestService_Process_NilOptions(t *testing.T) {
	svc, err := newServiceForTest()
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Process("/tmp/test.jpg", nil)

	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestService_Process_UnsupportedConversion(t *testing.T) {
	t.Skip("requires valid test image and external tools")
}

func TestService_Process_ResizeError(t *testing.T) {
	svc, err := newServiceForTest()
	if err != nil {
		t.Fatal(err)
	}

	svc.resizer.resizeErr = errors.New("resize failed")
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jpg")
	// Write minimal valid JPEG for metadata detection
	err = os.WriteFile(testFile, []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}, 0644)
	if err != nil {
		t.Fatal(err)
	}

	opt, _ := processing.NewOptions(image.FormatJPEG, 100, 80, false, nil)
	_, err = svc.Process(testFile, opt)

	if err == nil {
		t.Error("expected error from resize")
	}
}

func TestService_Process_CompressError(t *testing.T) {
	svc, err := newServiceForTest()
	if err != nil {
		t.Fatal(err)
	}

	svc.compressor.compressErr = errors.New("compress failed")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jpg")
	err = os.WriteFile(testFile, []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}, 0644)
	if err != nil {
		t.Fatal(err)
	}

	opt, _ := processing.NewOptions(image.FormatJPEG, 0, 80, false, nil)
	_, err = svc.Process(testFile, opt)

	if err == nil {
		t.Error("expected error from compress")
	}
}

func TestResizeDimensions_Additional(t *testing.T) {
	tests := []struct {
		name                 string
		maxDim               int
		origW, origH         int
		expectedW, expectedH int
	}{
		{"smaller than max", 1000, 800, 600, 800, 600},
		{"resize width", 1000, 2000, 600, 1000, 300},
		{"resize height", 1000, 800, 2000, 400, 1000},
		{"equal to max", 1000, 1000, 1000, 1000, 1000},
		{"zero max dimension", 0, 800, 600, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := resizeDimensions(tt.maxDim, tt.origW, tt.origH)
			if w != tt.expectedW || h != tt.expectedH {
				t.Errorf("resizeDimensions(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.maxDim, tt.origW, tt.origH, w, h, tt.expectedW, tt.expectedH)
			}
		})
	}
}

func TestNeedToConvert_AdditionalCases(t *testing.T) {
	tests := []struct {
		name     string
		from     image.Format
		to       image.Format
		expected bool
	}{
		{"png -> webp (compression)", image.FormatPNG, image.FormatWEBP, false},
		{"webp -> png (not supported)", image.FormatWEBP, image.FormatPNG, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needToConvert(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("needToConvert(%q -> %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestService_Process_InvalidCompressionRatio(t *testing.T) {
	_, err := processing.NewOptions(image.FormatJPEG, 0, 0, false, nil) // 0 is invalid
	if err == nil {
		t.Error("expected error for invalid compression ratio")
	}
}

func TestService_Process_InvalidMaxDimension(t *testing.T) {
	_, err := processing.NewOptions(image.FormatJPEG, -10, 80, false, nil) // negative is invalid
	if err == nil {
		t.Error("expected error for negative max dimension")
	}
}

func TestService_Process_Integration(t *testing.T) {
	t.Skip("Integration test - requires external tools (ImageMagick, jpegoptim, etc.)")

	svc, err := NewService()
	if err != nil {
		t.Skipf("skipping: external tools not available: %v", err)
	}

	// Would need real test images
	_ = svc
}
