package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"saveme/internal/core"
)

type Protector interface {
	Set(path string, enabled bool) error
}

type Store struct {
	mu         sync.Mutex
	dir        string
	protect    Protector
	generation uint64
}

type record struct {
	Generation uint64     `json:"generation"`
	State      core.State `json:"state"`
	Checksum   string     `json:"checksum"`
}

const recordLimit = 1024 * 1024

func New(dir string, protect Protector) *Store { return &Store{dir: dir, protect: protect} }

func (s *Store) paths() []string {
	const primary, backup = "state.json", "state.bak.json"
	return []string{filepath.Join(s.dir, primary), filepath.Join(s.dir, backup)}
}

func checksum(state core.State, generation uint64) (string, error) {
	const empty = ""
	type payload struct {
		Generation uint64     `json:"generation"`
		State      core.State `json:"state"`
	}
	data, err := json.Marshal(payload{Generation: generation, State: state})
	if err != nil {
		return empty, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func readRecord(path string) (record, error) {
	info, err := os.Stat(path)
	if err != nil {
		return record{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > recordLimit {
		return record{}, core.ErrStorage
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return record{}, err
	}
	var result record
	if err := json.Unmarshal(data, &result); err != nil {
		return record{}, err
	}
	sum, err := checksum(result.State, result.Generation)
	if err != nil || sum != result.Checksum {
		return record{}, core.ErrStorage
	}
	if err := result.State.Validate(); err != nil {
		return record{}, err
	}
	return result, nil
}

func (s *Store) Load() (core.State, error) {
	const zero, copies, failure = 0, 2, "no valid state copy"
	s.mu.Lock()
	defer s.mu.Unlock()
	missing, valid := zero, false
	var best record
	for _, path := range s.paths() {
		candidate, err := readRecord(path)
		if errors.Is(err, os.ErrNotExist) {
			missing++
		}
		if err == nil && (!valid || candidate.Generation > best.Generation) {
			best, valid = candidate, true
		}
	}
	if valid {
		s.generation = best.Generation
		return best.State.Clone(), nil
	}
	if missing == copies {
		return core.EmptyState(), nil
	}
	return core.State{}, errors.New(failure)
}

func (s *Store) Save(state core.State) error {
	const mode, zero, one = 0700, 0, 1
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := state.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, mode); err != nil {
		return err
	}
	generation := s.generation + one
	sum, err := checksum(state, generation)
	if err != nil {
		return err
	}
	next := record{Generation: generation, State: state, Checksum: sum}
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if len(data) > recordLimit {
		return core.ErrStorage
	}
	successes := zero
	var failures []error
	for _, path := range s.paths() {
		if err := s.replace(path, data); err != nil {
			failures = append(failures, err)
		} else {
			successes++
		}
	}
	if successes == zero {
		return errors.Join(failures...)
	}
	s.generation = next.Generation
	return errors.Join(failures...)
}

func (s *Store) replace(path string, data []byte) error {
	const pattern, label, mode = ".state-*", "protect state: %w", 0600
	file, err := os.CreateTemp(s.dir, pattern)
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if s.protect != nil {
		if err := s.protect.Set(path, false); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		defer s.protect.Set(path, true)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	if s.protect != nil {
		if err := s.protect.Set(path, true); err != nil {
			return fmt.Errorf(label, err)
		}
	}
	dir, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
