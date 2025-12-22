package todo

import (
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("todo not found")
var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(title string, done bool) (Todo, error) {
	if strings.TrimSpace(title) == "" {
		return Todo{}, ErrInvalidInput
	}

	t := s.store.Create(title, done, time.Now().Unix())
	return t, nil
}

func (s *Service) Get(id int64) (Todo, error) {
	if id <= 0 {
		return Todo{}, ErrInvalidInput
	}

	t, ok := s.store.Get(id)
	if !ok {
		return Todo{}, ErrNotFound
	}

	return t, nil
}

func (s *Service) List(limit, offset int) ([]Todo, error) {
	if limit < 0 || offset < 0 {
		return nil, ErrInvalidInput
	}

	return s.store.List(limit, offset), nil
}
