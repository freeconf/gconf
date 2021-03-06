package node

import "github.com/freeconf/gconf/c2"

type MaxNode struct {
	Count int
	Max   int
}

func (self MaxNode) CheckContainerPreConstraints(r *ChildRequest) (bool, error) {
	if r.IsNavigation() {
		return true, nil
	}
	self.Count++
	if self.Count > self.Max {
		return false, c2.NewErrC("response for request too large", 413)
	}
	return true, nil
}
