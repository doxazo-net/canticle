package secrets

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// MemoryStore is an in-memory Store for tests and single-process use, mirroring
// auth.MemoryStore. Values are held in plaintext in a map (there is no at-rest
// surface to protect in memory), so it exercises the same Store contract as
// SQLStore without a key or database.
type MemoryStore struct {
	mu        sync.RWMutex
	secrets   map[string]string
	updatedAt map[string]string
}

// NewMemoryStore returns an empty in-memory secret store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{secrets: map[string]string{}, updatedAt: map[string]string{}}
}

// Set stores plaintext under name, overwriting any existing value, and refreshes
// the name's updated_at to now (UTC, RFC3339-like, matching the SQL store format).
func (s *MemoryStore) Set(ctx context.Context, name, plaintext string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return errors.New("secrets: name must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[name] = plaintext
	s.updatedAt[name] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	return nil
}

// Get returns the plaintext for name; ok is false when absent.
func (s *MemoryStore) Get(ctx context.Context, name string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.secrets[name]
	return v, ok, nil
}

// Delete removes name; deleting an absent name is a no-op.
func (s *MemoryStore) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, name)
	delete(s.updatedAt, name)
	return nil
}

// List returns name + updated_at for every stored secret, ordered by name. It
// never returns values, mirroring SQLStore.List.
func (s *MemoryStore) List(ctx context.Context) ([]SecretInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SecretInfo, 0, len(s.secrets))
	for name := range s.secrets {
		out = append(out, SecretInfo{Name: name, UpdatedAt: s.updatedAt[name]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
