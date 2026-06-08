package main

import (
	"fmt"
	"sync"
	"time"
)

type Store struct {
	mu     sync.RWMutex
	config *Config
}

func NewStore() (*Store, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return &Store{config: cfg}, nil
}

func (s *Store) GetRules() []Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Rule, len(s.config.Rules))
	copy(result, s.config.Rules)
	return result
}

func (s *Store) GetRule(id string) (*Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.config.Rules {
		if s.config.Rules[i].ID == id {
			r := s.config.Rules[i]
			return &r, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", id)
}

func (s *Store) AddRule(r Rule) (Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = generateID()
	}
	r.CreatedAt = time.Now().Format(time.RFC3339)
	s.config.Rules = append(s.config.Rules, r)
	if err := saveConfig(s.config); err != nil {
		return Rule{}, err
	}
	return r, nil
}

func (s *Store) UpdateRule(id string, r Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.config.Rules {
		if s.config.Rules[i].ID == id {
			r.ID = id
			r.CreatedAt = s.config.Rules[i].CreatedAt
			s.config.Rules[i] = r
			return saveConfig(s.config)
		}
	}
	return fmt.Errorf("rule not found: %s", id)
}

func (s *Store) DeleteRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.config.Rules {
		if s.config.Rules[i].ID == id {
			s.config.Rules = append(s.config.Rules[:i], s.config.Rules[i+1:]...)
			return saveConfig(s.config)
		}
	}
	return fmt.Errorf("rule not found: %s", id)
}

func (s *Store) ToggleRule(id string) (*Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.config.Rules {
		if s.config.Rules[i].ID == id {
			s.config.Rules[i].Enabled = !s.config.Rules[i].Enabled
			r := s.config.Rules[i]
			if err := saveConfig(s.config); err != nil {
				return nil, err
			}
			return &r, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", id)
}

func (s *Store) ProxyPort() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.ProxyPort
}

func (s *Store) WebPort() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.WebPort
}
