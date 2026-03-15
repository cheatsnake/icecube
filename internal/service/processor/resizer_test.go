package processor

import "testing"

func TestResizeDimensions(t *testing.T) {
	tests := []struct {
		name           string
		maxDimension   int
		originalWidth  int
		originalHeight int
		expectedWidth  int
		expectedHeight int
	}{
		// No resize needed - already smaller than max
		{"already small", 100, 80, 60, 80, 60},
		{"exactly max width", 100, 100, 50, 100, 50},
		{"exactly max height", 100, 50, 100, 50, 100},

		// Resize by width (width > height)
		{"resize by width", 100, 200, 100, 100, 50},
		{"resize by width large", 800, 1920, 1080, 800, 450},

		// Resize by height (height > width)
		{"resize by height", 100, 100, 200, 50, 100},
		{"resize by height large", 800, 1080, 1920, 450, 800},

		// Square images
		{"square resize", 100, 200, 200, 100, 100},
		{"square no resize", 200, 200, 200, 200, 200},

		// Edge cases
		{"zero max dimension", 0, 100, 100, 0, 0},
		{"zero dimensions", 100, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := resizeDimensions(tt.maxDimension, tt.originalWidth, tt.originalHeight)
			if width != tt.expectedWidth || height != tt.expectedHeight {
				t.Errorf("resizeDimensions(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.maxDimension, tt.originalWidth, tt.originalHeight,
					width, height, tt.expectedWidth, tt.expectedHeight)
			}
		})
	}
}
