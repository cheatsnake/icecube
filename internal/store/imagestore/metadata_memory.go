package imagestore

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/pkg/errs"
)

type metadataStoreMemory struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	variants map[string]*image.Variant
}

func newMetadataStoreMemory(logger *slog.Logger) *metadataStoreMemory {
	return &metadataStoreMemory{
		logger:   logger,
		variants: make(map[string]*image.Variant),
	}
}

// NewTestMetadataStoreMemory creates a new in-memory metadata store for testing
func NewTestMetadataStoreMemory(logger *slog.Logger) *metadataStoreMemory {
	return newMetadataStoreMemory(logger)
}

func (s *metadataStoreMemory) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	variant, exists := s.variants[id]
	if !exists {
		return nil, errors.Join(errs.ErrNotFound, errors.New("variant not found: "+id))
	}

	return &image.Variant{
		ID:           variant.ID,
		OriginalName: variant.OriginalName,
		Format:       variant.Format,
		Width:        variant.Width,
		Height:       variant.Height,
		ByteSize:     variant.ByteSize,
	}, nil
}

func (s *metadataStoreMemory) GetMetadataByIDs(ctx context.Context, ids []string) ([]*image.Variant, error) {
	if len(ids) == 0 {
		return []*image.Variant{}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	variants := make([]*image.Variant, 0, len(ids))
	for _, id := range ids {
		if variant, exists := s.variants[id]; exists {
			variants = append(variants, &image.Variant{
				ID:           variant.ID,
				OriginalName: variant.OriginalName,
				Format:       variant.Format,
				Width:        variant.Width,
				Height:       variant.Height,
				ByteSize:     variant.ByteSize,
			})
		}
	}

	return variants, nil
}

func (s *metadataStoreMemory) AddMetadata(ctx context.Context, metadata *image.Variant) error {
	if metadata == nil {
		return errors.Join(errs.ErrInvalidInput, errors.New("metadata cannot be nil"))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.variants[metadata.ID] = &image.Variant{
		ID:           metadata.ID,
		OriginalName: metadata.OriginalName,
		Format:       metadata.Format,
		Width:        metadata.Width,
		Height:       metadata.Height,
		ByteSize:     metadata.ByteSize,
	}

	s.logger.Debug("Metadata added", "id", metadata.ID)
	return nil
}

func (s *metadataStoreMemory) DeleteMetadataByID(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.variants[id]; !exists {
		return errors.Join(errs.ErrNotFound, errors.New("variant not found: "+id))
	}

	delete(s.variants, id)
	return nil
}

func (s *metadataStoreMemory) DeleteMetadataByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.variants, id)
	}

	return nil
}
