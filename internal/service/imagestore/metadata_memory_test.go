package imagestore

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cheatsnake/icecube/internal/domain/image"
)

func TestNewMetadataStoreMemory(t *testing.T) {
	store := NewMetadataStoreMemory(slog.Default())
	if store == nil {
		t.Error("NewMetadataStoreMemory(slog.Default()) returned nil")
	}
}

func TestAddMetadata(t *testing.T) {
	store := NewMetadataStoreMemory(slog.Default())
	ctx := context.Background()

	variant, _ := image.NewVariant("test-id", "photo.jpg", image.FormatJPEG, 100, 200, 5000)

	err := store.AddMetadata(ctx, variant)
	if err != nil {
		t.Errorf("AddMetadata() error = %v", err)
	}

	// Add duplicate should overwrite
	err = store.AddMetadata(ctx, variant)
	if err != nil {
		t.Errorf("AddMetadata() duplicate error = %v", err)
	}

	// Add nil should fail
	err = store.AddMetadata(ctx, nil)
	if err == nil {
		t.Error("AddMetadata() should return error for nil")
	}
}

func TestGetMetadataByID(t *testing.T) {
	store := NewMetadataStoreMemory(slog.Default())
	ctx := context.Background()

	// Get non-existing
	_, err := store.GetMetadataByID(ctx, "non-existing")
	if err == nil {
		t.Error("GetMetadataByID() should return error for non-existing")
	}

	// Add and get
	variant, _ := image.NewVariant("test-id", "photo.jpg", image.FormatJPEG, 100, 200, 5000)
	store.AddMetadata(ctx, variant)

	got, err := store.GetMetadataByID(ctx, "test-id")
	if err != nil {
		t.Errorf("GetMetadataByID() error = %v", err)
	}
	if got.ID != "test-id" {
		t.Errorf("GetMetadataByID() ID = %v, want 'test-id'", got.ID)
	}
	if got.Width != 100 {
		t.Errorf("GetMetadataByID() Width = %v, want 100", got.Width)
	}
}

func TestGetMetadataByIDs(t *testing.T) {
	store := NewMetadataStoreMemory(slog.Default())
	ctx := context.Background()

	// Empty list
	result, err := store.GetMetadataByIDs(ctx, []string{})
	if err != nil {
		t.Errorf("GetMetadataByIDs() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("GetMetadataByIDs() len = %v, want 0", len(result))
	}

	// Add variants
	v1, _ := image.NewVariant("id1", "photo1.jpg", image.FormatJPEG, 100, 100, 1000)
	v2, _ := image.NewVariant("id2", "photo2.png", image.FormatPNG, 200, 200, 2000)
	v3, _ := image.NewVariant("id3", "photo3.webp", image.FormatWEBP, 300, 300, 3000)
	store.AddMetadata(ctx, v1)
	store.AddMetadata(ctx, v2)
	store.AddMetadata(ctx, v3)

	// Get multiple
	result, err = store.GetMetadataByIDs(ctx, []string{"id1", "id2"})
	if err != nil {
		t.Errorf("GetMetadataByIDs() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("GetMetadataByIDs() len = %v, want 2", len(result))
	}

	// Get with non-existing
	result, err = store.GetMetadataByIDs(ctx, []string{"id1", "non-existing", "id2"})
	if err != nil {
		t.Errorf("GetMetadataByIDs() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("GetMetadataByIDs() len = %v, want 2 (skip non-existing)", len(result))
	}
}

func TestDeleteMetadataByID(t *testing.T) {
	store := NewMetadataStoreMemory(slog.Default())
	ctx := context.Background()

	variant, _ := image.NewVariant("test-id", "photo.jpg", image.FormatJPEG, 100, 200, 5000)
	store.AddMetadata(ctx, variant)

	// Delete
	err := store.DeleteMetadataByID(ctx, "test-id")
	if err != nil {
		t.Errorf("DeleteMetadataByID() error = %v", err)
	}

	// Verify deleted
	_, err = store.GetMetadataByID(ctx, "test-id")
	if err == nil {
		t.Error("GetMetadataByID() should return error after delete")
	}

	// Delete non-existing
	err = store.DeleteMetadataByID(ctx, "non-existing")
	if err == nil {
		t.Error("DeleteMetadataByID() should return error for non-existing")
	}
}

func TestDeleteMetadataByIDs(t *testing.T) {
	store := NewMetadataStoreMemory(slog.Default())
	ctx := context.Background()

	v1, _ := image.NewVariant("id1", "photo1.jpg", image.FormatJPEG, 100, 100, 1000)
	v2, _ := image.NewVariant("id2", "photo2.png", image.FormatPNG, 200, 200, 2000)
	store.AddMetadata(ctx, v1)
	store.AddMetadata(ctx, v2)

	// Delete multiple
	err := store.DeleteMetadataByIDs(ctx, []string{"id1", "id2"})
	if err != nil {
		t.Errorf("DeleteMetadataByIDs() error = %v", err)
	}

	// Verify deleted
	result, _ := store.GetMetadataByIDs(ctx, []string{"id1", "id2"})
	if len(result) != 0 {
		t.Error("All variants should be deleted")
	}

	// Empty list should not error
	err = store.DeleteMetadataByIDs(ctx, []string{})
	if err != nil {
		t.Errorf("DeleteMetadataByIDs() empty error = %v", err)
	}
}
