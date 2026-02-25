package imagestore

import (
	"context"
	"fmt"
	"sync"

	"github.com/cheatsnake/icm/internal/domain/image"
)

type metadataStoreMemory struct {
	mu       sync.RWMutex
	variants map[string]*image.Variant
}

func newMetadataStoreMemory() (*metadataStoreMemory, error) {
	return &metadataStoreMemory{
		variants: make(map[string]*image.Variant),
	}, nil
}

func (s *metadataStoreMemory) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	variant, exists := s.variants[id]
	if !exists {
		return nil, fmt.Errorf("variant not found: %s", id)
	}

	return &image.Variant{
		ID:       variant.ID,
		Format:   variant.Format,
		Width:    variant.Width,
		Height:   variant.Height,
		ByteSize: variant.ByteSize,
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
				ID:       variant.ID,
				Format:   variant.Format,
				Width:    variant.Width,
				Height:   variant.Height,
				ByteSize: variant.ByteSize,
			})
		}
	}

	return variants, nil
}

func (s *metadataStoreMemory) AddMetadata(ctx context.Context, metadata *image.Variant) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.variants[metadata.ID] = &image.Variant{
		ID:       metadata.ID,
		Format:   metadata.Format,
		Width:    metadata.Width,
		Height:   metadata.Height,
		ByteSize: metadata.ByteSize,
	}

	return nil
}

func (s *metadataStoreMemory) DeleteMetadataByID(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.variants[id]; !exists {
		return fmt.Errorf("variant not found: %s", id)
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
