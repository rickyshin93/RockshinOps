// Package trayimage draws menu bar readings as one template image.
package trayimage

import (
	"slices"
	"sync"
)

// Item is one SF Symbol followed by its reading.
type Item struct {
	Symbol string
	Text   string
}

var rendered imageCache

// Render reuses the last template PNG when its symbols and displayed values
// have not changed. The cache is bounded to one image and shared with previews.
func Render(items []Item) ([]byte, error) { return rendered.get(items, renderNative) }

type imageCache struct {
	mu    sync.Mutex
	items []Item
	image []byte
}

func (c *imageCache) get(items []Item, draw func([]Item) ([]byte, error)) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.image != nil && slices.Equal(c.items, items) {
		return c.image, nil
	}
	image, err := draw(items)
	if err != nil {
		return nil, err
	}
	c.items, c.image = slices.Clone(items), image
	return image, nil
}
