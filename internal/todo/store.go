package todo

import "sync"

type Store struct {
	mu     sync.Mutex
	nextID int64
	items  []Todo
}

func NewStore() *Store {
	return &Store{
		nextID: 1,
		items:  make([]Todo, 0),
	}
}

func (s *Store) Create(title string, done bool, createdAt int64) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Todo{
		ID:        s.nextID,
		Title:     title,
		Done:      done,
		CreatedAt: createdAt,
	}
	s.nextID++
	s.items = append(s.items, t)

	return t
}

func (s *Store) Get(id int64) (Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}

	return Todo{}, false
}

func (s *Store) List(limit, offset int) []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset < 0 || offset >= len(s.items) {
		return []Todo{}
	}

	end := len(s.items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	out := make([]Todo, end-offset)
	copy(out, s.items[offset:end])
	return out
}
