package cards

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Store é a base local de cartas, persistida em um único JSON.
// Chave: ID canônico da carta.
type Store struct {
	Cards map[string]*Card `json:"cards"`
}

func NewStore() *Store {
	return &Store{Cards: make(map[string]*Card)}
}

// Load carrega a base do disco. Arquivo inexistente devolve base vazia.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewStore(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	s := NewStore()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return s, nil
}

// Save grava a base em disco de forma atômica (escreve em .tmp e renomeia).
func (s *Store) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}

// Put insere ou substitui uma carta.
func (s *Store) Put(c *Card) { s.Cards[c.ID] = c }

// SetIDs devolve os IDs de sets presentes na base, ordenados.
func (s *Store) SetIDs() []string {
	seen := make(map[string]bool)
	for _, c := range s.Cards {
		seen[c.SetID] = true
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
