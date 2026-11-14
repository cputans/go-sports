package dataset

import "sync"

// Cache holds one value per season, loaded at most once per process.
type Cache[T any] struct {
	mu      sync.Mutex
	entries map[int]*entry[T]
}

type entry[T any] struct {
	once sync.Once
	data *T
	err  error
}

// Load returns the cached data for a season, calling load on first use. A
// failed load is cached too, so a missing season isn't re-downloaded for every
// team and game.
func (c *Cache[T]) Load(year int, load func(year int) (*T, error)) (*T, error) {
	c.mu.Lock()
	if c.entries == nil {
		c.entries = map[int]*entry[T]{}
	}
	e, ok := c.entries[year]
	if !ok {
		e = &entry[T]{}
		c.entries[year] = e
	}
	c.mu.Unlock()

	e.once.Do(func() { e.data, e.err = load(year) })
	return e.data, e.err
}

// Reset drops everything cached.
func (c *Cache[T]) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = nil
}
