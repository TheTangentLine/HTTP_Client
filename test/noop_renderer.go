package test

import (
	"github.com/thetangentline/httpcl/internal/stats"
)

// noopRenderer implements ui.Renderer and discards all output (for tests).
type noopRenderer struct{}

func (n *noopRenderer) Render(snap stats.Snapshot) {}

func (n *noopRenderer) RenderFinal(snap stats.Snapshot) {}

// NewNoopRenderer returns a renderer that does nothing (used by integration tests).
func NewNoopRenderer() *noopRenderer {
	return &noopRenderer{}
}
