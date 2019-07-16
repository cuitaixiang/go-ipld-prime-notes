package selector

import (
	"fmt"

	ipld "github.com/ipld/go-ipld-prime"
)

// ExploreIndex traverses a specific index in a list, and applies a next
// selector to the reached node.
type ExploreIndex struct {
	next     Selector      // selector for element we're interested in
	interest []PathSegment // index of element we're interested in
}

// Interests for ExploreIndex is just the index specified by the selector node
func (s ExploreIndex) Interests() []PathSegment {
	return s.interest
}

// Explore returns the node's selector if
// the path matches the index the index for this selector or nil if not
func (s ExploreIndex) Explore(n ipld.Node, p PathSegment) Selector {
	if n.ReprKind() != ipld.ReprKind_List {
		return nil
	}
	expectedIndex, expectedErr := p.Index()
	actualIndex, actualErr := s.interest[0].Index()
	if expectedErr != nil || actualErr != nil || expectedIndex != actualIndex {
		return nil
	}
	return s.next
}

// Decide always returns false because this is not a matcher
func (s ExploreIndex) Decide(n ipld.Node) bool {
	return false
}

// ParseExploreIndex assembles a Selector
// from a ExploreIndex selector node
func ParseExploreIndex(n ipld.Node) (Selector, error) {
	if n.ReprKind() != ipld.ReprKind_Map {
		return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map")
	}
	indexNode, err := n.TraverseField(indexKey)
	if err != nil {
		return nil, fmt.Errorf("selector spec parse rejected: index field must be present in ExploreIndex selector")
	}
	indexValue, err := indexNode.AsInt()
	if err != nil {
		return nil, fmt.Errorf("selector spec parse rejected: index field must be a number in ExploreIndex selector")
	}
	next, err := n.TraverseField(nextSelectorKey)
	if err != nil {
		return nil, fmt.Errorf("selector spec parse rejected: next field must be present in ExploreIndex selector")
	}
	selector, err := ParseSelector(next)
	if err != nil {
		return nil, err
	}
	return ExploreIndex{selector, []PathSegment{PathSegmentInt{I: indexValue}}}, nil
}
