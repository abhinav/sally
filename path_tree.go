package main

import (
	"strings"
)

// pathTree holds values in a tree-like hierarchy defined by /-separated paths
// (e.g. import paths).
// Values defined at a path cascade down to all descendants
// unless a descendant has its own value specified.
//
// The zero value of pathTree is safe for use.
type pathTree[T any] struct {
	root pathTreeNode[T]

	// Approximate number of nodes with explicit values.
	// This is not exact.
	// It is used to optimize the ListByPath method.
	countHint int
}

// Set sets the value in the tree at the given path.
// All descendants of the path will inherit the value
// unless a value is set for them explicitly.
func (t *pathTree[T]) Set(path string, value T) {
	t.countHint++
	t.root.set(path, &value)
}

// Lookup retrieves the value for the given path.
// If the path doesn't have an explicit value set,
// the value for the closest ancestor with a value is returned.
// If no value is set for the path or any of its ancestors,
// this returns false.
//
// The returned path is the path for which a value was found.
// It may be different from the path passed to Lookup.
func (t *pathTree[T]) Lookup(path string) (found string, v T, ok bool) {
	n := t.root.lookup(path)
	if n == nil || n.value == nil {
		return found, v, false
	}
	return n.path, *n.value, true
}

// ListByPath returns a map of all values in the tree
// that are descendants of the given path.
// The returned map is keyed by the path for each value.
func (t *pathTree[T]) ListByPath(path string) map[string]T {
	n := &t.root
	if path != "" {
		n = n.get(path)
	}
	if n == nil {
		return nil
	}

	items := make(map[string]T, t.countHint)
	n.listByPath(items)
	return items
}

// pathTreeNode is a single node in a pathTree.
// Don't use this directly.
type pathTreeNode[T any] struct {
	// Single component of the path to this node.
	// E.g. "bar" in "foo/bar/baz".
	// Empty for the root node.
	name string

	// Full path to this node from the root.
	path string

	// Value for this node.
	// Non-nil only if this node has an explicit value assigned to it.
	value *T

	// Direct descendants of this node, keyed by name.
	children map[string]*pathTreeNode[T]
}

// Sets a descendant's value, given a relative path.
// Empty path sets value for this node.
//
// Invariant: value must not be nil.
func (n *pathTreeNode[T]) set(path string, value *T) {
	// origPath[:curPathLen] is the path to the node we're setting.
	// We use this to set child.path.
	// This is cheaper than allocating a new path for each node.
	origPath := path
	var curPathLen int

	// Descends the tree one path component at a time,
	// creating nodes as needed.
	for len(path) > 0 {
		head, tail := pathTakeFirst(path)

		// If we're looking at "bar" in "foo/bar/baz",
		// curPathLen is the length of "foo".
		// Add to make it "foo/bar".
		if curPathLen > 0 {
			curPathLen++ // preceding '/'
		}
		curPathLen += len(head)

		if n.children == nil {
			n.children = make(map[string]*pathTreeNode[T])
		}

		ch, ok := n.children[head]
		if !ok {
			ch = &pathTreeNode[T]{
				name: head,
				path: origPath[:curPathLen],
			}
			n.children[head] = ch
		}

		n = ch
		path = tail
	}
	n.value = value
}

// Finds and returns the deepest node on the given path with a non-nil value.
// Returns nil if nothing in that path has a non-nil value.
func (n *pathTreeNode[T]) lookup(path string) (result *pathTreeNode[T]) {
	var last *pathTreeNode[T] // last node with a non-nil value
	for n != nil {
		if n.value != nil {
			last = n
		}
		if len(path) == 0 {
			break
		}

		head, tail := pathTakeFirst(path)
		n = n.children[head]
		path = tail
	}
	return last
}

// Gets the node at the given path.
// Returns nil if a node doesn't exist at that path.
func (n *pathTreeNode[T]) get(path string) *pathTreeNode[T] {
	for n != nil && len(path) > 0 {
		head, tail := pathTakeFirst(path)
		n = n.children[head]
		path = tail
	}
	return n
}

// Puts all values in the subtree rooted at this node
// into the given map, keyed by path.
func (n *pathTreeNode[T]) listByPath(items map[string]T) {
	next := []*pathTreeNode[T]{n}
	for len(next) > 0 {
		// Treat the slice as a stack.
		n := next[len(next)-1]
		next = next[:len(next)-1]

		if n.value != nil {
			items[n.path] = *n.value
		}

		for _, ch := range n.children {
			next = append(next, ch)
		}
	}
}

// Takes the first component of a path, returning it and the rest.
//
//	pathTakeFirst("foo/bar/baz")
//	// => ("foo", "bar/baz")
func pathTakeFirst(p string) (head, tail string) {
	head, tail = p, ""
	if idx := strings.IndexByte(p, '/'); idx >= 0 {
		head, tail = p[:idx], p[idx+1:]
	}

	return head, tail
}
