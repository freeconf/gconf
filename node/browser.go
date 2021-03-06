package node

import "github.com/freeconf/gconf/meta"
import "context"

// Browser is a handle to a data source and starting point for interfacing with any freeconf enabled interface.
// It's the starting point to the top-most selection, or the Root().
type Browser struct {
	Meta *meta.Module

	// TODO: is this used?
	Triggers *TriggerTable

	src func() Node
}

// Root is top-most selection.  From here you can use Find to navigate to other parts of
// application or any of the Insert command to start getting data in or out.
func (self *Browser) Root() Selection {
	return Selection{
		Browser:     self,
		Path:        &Path{meta: self.Meta},
		Node:        self.src(),
		Constraints: baseConstraints(),
		Context:     context.Background(),
	}
}

// Root is top-most selection.  From here you can use Find to navigate to other parts of
// application or any of the Insert command to start getting data in or out.
func (self *Browser) RootWithContext(ctx context.Context) Selection {
	return Selection{
		Browser:     self,
		Path:        &Path{meta: self.Meta},
		Node:        self.src(),
		Constraints: baseConstraints(),
		Context:     ctx,
	}
}

func baseConstraints() *Constraints {
	c := &Constraints{}
	c.AddConstraint("~when", 100, 0, CheckWhen{})
	return c
}

// NewBrowserSource unites a model (MetaList) with a data source (Node).  Here the node instance
// is requested for each browse operation allowing the node state to be fresh for each request.
func NewBrowserSource(m *meta.Module, src func() Node) *Browser {
	return &Browser{
		Meta:     m,
		Triggers: NewTriggerTable(),
		src:      src,
	}
}

// NewBrowser  obviously does not resolve the source node for each new selection
// so the state of at least the root node is shared for all subsequent operations.
// In short, either do not keep a copy of this very browser for very long or know
// what you're doing
func NewBrowser(m *meta.Module, n Node) *Browser {
	return &Browser{
		Meta:     m,
		Triggers: NewTriggerTable(),
		src: func() Node {
			return n
		},
	}
}
