package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestPathTree_empty(t *testing.T) {
	t.Parallel()

	var tree pathTree[int]

	_, _, ok := tree.Lookup("")
	assert.False(t, ok)

	_, _, ok = tree.Lookup("foo")
	assert.False(t, ok)

	assert.Empty(t, tree.ListByPath(""))
}

func TestPathTree_simple(t *testing.T) {
	t.Parallel()

	var tree pathTree[int]
	mustHave := func(path, match string, want int) {
		t.Helper()

		found, v, ok := tree.Lookup(path)
		require.True(t, ok, "path %q", path)
		assert.Equal(t, match, found, "path %q", path)
		assert.Equal(t, v, want, "path %q", path)
	}

	mustNotHave := func(path string) {
		t.Helper()

		_, _, ok := tree.Lookup(path)
		require.False(t, ok, "path %q", path)
	}

	mustList := func(path string, pairs ...any) {
		t.Helper()

		require.True(t, len(pairs)%2 == 0, "pairs must be even")
		var want map[string]int
		if len(pairs) > 0 {
			want = make(map[string]int, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				k := pairs[i].(string)
				v := pairs[i+1].(int)
				want[k] = v
			}
		}

		assert.Equal(t, want, tree.ListByPath(path), "path %q", path)
	}

	tree.Set("foo", 10)
	t.Run("single", func(t *testing.T) {
		mustHave("foo", "foo", 10)
		mustHave("foo/bar", "foo", 10)
		mustHave("foo/bar/baz", "foo", 10)
		mustNotHave("")
		mustNotHave("bar")
		mustNotHave("bar/baz")

		t.Run("list", func(t *testing.T) {
			mustList("", "foo", 10)
			mustList("foo", "foo", 10)
			mustList("foo/bar")
		})
	})

	// Override a descendant value.
	t.Run("descendant", func(t *testing.T) {
		tree.Set("foo/bar", 20)
		mustHave("foo", "foo", 10)
		mustHave("foo/bar", "foo/bar", 20)
		mustHave("foo/bar/baz", "foo/bar", 20)

		t.Run("list", func(t *testing.T) {
			mustList("", "foo", 10, "foo/bar", 20)
			mustList("foo", "foo", 10, "foo/bar", 20)
			mustList("foo/bar", "foo/bar", 20)
			mustList("foo/bar/baz")
		})
	})

	// Add a sibling.
	t.Run("sibling", func(t *testing.T) {
		tree.Set("bar", 30)
		mustHave("bar", "bar", 30)
		mustHave("bar/baz", "bar", 30)

		t.Run("list", func(t *testing.T) {
			mustList("", "foo", 10, "foo/bar", 20, "bar", 30)
			mustList("foo", "foo", 10, "foo/bar", 20)
			mustList("bar", "bar", 30)
			mustList("bar/baz")
		})
	})
}

func TestPathTree_rapid(t *testing.T) {
	t.Parallel()

	rapid.Check(t, rapid.Run[*pathTreeStateMachine]())
}

// pathTreeStateMachine is a rapid.StateMachine.
// It generates random path trees and performs random operations on them.
type pathTreeStateMachine struct {
	tree pathTree[int]

	// items is used to verify the state of the tree.
	items map[string]int // path => value
	paths []string       // all paths in items
}

func (m *pathTreeStateMachine) drawPath(t *rapid.T) string {
	return rapid.StringMatching(`[a-zA-Z0-9_]+(/[a-zA-Z0-9_]+)*`).Draw(t, "path")
}

// Init initializes the state machine.
func (m *pathTreeStateMachine) Init(t *rapid.T) {
	n := rapid.IntRange(0, 100).Draw(t, "n")
	m.items = make(map[string]int, n)
	for i := 0; i < n; i++ {
		m.Set(t)
	}
}

func (m *pathTreeStateMachine) Set(t *rapid.T) {
	path := m.drawPath(t)
	value := rapid.Int().Draw(t, "value")

	m.tree.Set(path, value)
	if _, ok := m.items[path]; !ok {
		// Don't add duplicates to the list.
		m.paths = append(m.paths, path)
	}
	m.items[path] = value
}

func (m *pathTreeStateMachine) Lookup(t *rapid.T) {
	path := m.drawPath(t)
	gotPath, got, ok := m.tree.Lookup(path)

	// Expect an exact match if the path exists.
	if want, has := m.items[path]; has {
		require.True(t, ok, "value for %q not found", path)
		assert.Equal(t, path, gotPath, "matched path for %q is incorrect", path)
		assert.Equal(t, want, got, "value for %q is incorrect", path)
		return
	}

	// Otherwise, the value may or may not exist.
	// If it doesn't, we don't need to check anything else.
	// If it does, gotPath must be an ancestor of path.
	if !ok {
		return
	}

	assert.True(t, strings.HasPrefix(path, gotPath+"/"),
		"matched path %q is not an ancestor of %q", gotPath, path)
}

func (m *pathTreeStateMachine) LookupKnownChild(t *rapid.T) {
	if len(m.paths) == 0 {
		t.Skip()
	}
	parent := rapid.SampledFrom(m.paths).Draw(t, "path")
	child := parent + "/" + m.drawPath(t)
	if _, ok := m.items[child]; ok {
		// Unlikely, but random child might already exist in the tree.
		t.Skip()
	}

	want, ok := m.items[parent]
	require.True(t, ok, "parent %q not found", parent)

	gotPath, got, ok := m.tree.Lookup(child)
	require.True(t, ok, "child %q not found", child)
	assert.Equal(t, parent, gotPath, "matched path for child %q is incorrect", child)
	assert.Equal(t, want, got, "matched value for child %q is incorrect", child)
}

func (m *pathTreeStateMachine) ListKnown(t *rapid.T) {
	if len(m.paths) == 0 {
		t.Skip()
	}
	path := rapid.SampledFrom(m.paths).Draw(t, "path")

	want := make(map[string]int)
	for p, v := range m.items {
		if p == path || strings.HasPrefix(p, path+"/") {
			want[p] = v
		}
	}

	got := m.tree.ListByPath(path)
	assert.Equal(t, want, got, "ListByPath(%q)", path)
}

func (m *pathTreeStateMachine) ListAll(t *rapid.T) {
	if len(m.items) == 0 {
		t.Skip()
	}

	got := m.tree.ListByPath("")
	assert.Equal(t, m.items, got, `ListByPath("")`)
}

// Check verifies that the state machine is in a valid state.
func (m *pathTreeStateMachine) Check(t *rapid.T) {
	for path, want := range m.items {
		gotPath, got, ok := m.tree.Lookup(path)
		require.True(t, ok, "value for %q not found", path)
		assert.Equal(t, path, gotPath, "path for %q is incorrect", path)
		assert.Equal(t, want, got, "value for %q is incorrect", path)
	}
}

func BenchmarkPathTreeDeep(b *testing.B) {
	b.Run("100", func(b *testing.B) {
		benchmarkPathTreeDeep(b, 100)
	})
	b.Run("1000", func(b *testing.B) {
		benchmarkPathTreeDeep(b, 1000)
	})
}

func benchmarkPathTreeDeep(b *testing.B, N int) {
	var (
		tree    pathTree[int]
		curPath strings.Builder
	)
	paths := make([]string, 0, N)
	for i := 0; i < N; i++ {
		if curPath.Len() > 0 {
			curPath.WriteByte('/')
		}
		curPath.WriteString("a")

		path := curPath.String()
		paths = append(paths, path)
		tree.Set(path, i)
	}

	b.ResetTimer()

	b.Run("LookupDeepest", func(b *testing.B) {
		last := paths[len(paths)-1]
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _, ok := tree.Lookup(last)
				require.True(b, ok)
			}
		})
	})

	b.Run("LookupAll", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for i := 0; i < N; i++ {
					_, _, ok := tree.Lookup(paths[i])
					require.True(b, ok)
				}
			}
		})
	})

	b.Run("ListAll", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if len(tree.ListByPath("")) == 0 {
					b.Fatal("unexpected empty list")
				}
			}
		})
	})
}

func BenchmarkPathTreeWide(b *testing.B) {
	b.Run("100", func(b *testing.B) {
		benchmarkPathTreeWide(b, 100)
	})
	b.Run("1000", func(b *testing.B) {
		benchmarkPathTreeWide(b, 1000)
	})
}

func benchmarkPathTreeWide(b *testing.B, N int) {
	var tree pathTree[int]
	paths := make([]string, 0, N)
	for i := 0; i < N; i++ {
		path := strconv.Itoa(i)
		paths = append(paths, path)
		tree.Set(path, i)
	}

	b.ResetTimer()

	b.Run("LookupAll", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for i := 0; i < N; i++ {
					_, _, ok := tree.Lookup(paths[i])
					require.True(b, ok)
				}
			}
		})
	})

	b.Run("ListAll", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if len(tree.ListByPath("")) == 0 {
					b.Fatal("unexpected empty list")
				}
			}
		})
	})
}
