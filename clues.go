package clues

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/exp/maps"
)

// ---------------------------------------------------------------------------
// structure data storage and namespaces
// ---------------------------------------------------------------------------

const defaultNamespace = "clue_ns_default"

type syncValues struct {
	mu *sync.RWMutex
	m  map[string]any
}

func newValues() syncValues {
	return syncValues{
		mu: &sync.RWMutex{},
		m:  map[string]any{},
	}
}

func (sv syncValues) add(m map[string]any) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	maps.Copy(sv.m, m)
}

func (sv syncValues) data() values {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	vs := make(values)
	maps.Copy(vs, sv.m)
	return vs
}

type values map[string]any

func (vs values) Slice() []any {
	s := make([]any, 0, 2*len(vs))

	for k, v := range vs {
		s = append(s, k, v)
	}

	return s
}

// outer map tracks namespaces
// inner map tracks key/value pairs
type namespacedClues struct {
	mu *sync.RWMutex
	m  map[string]syncValues
}

func newClueMap() namespacedClues {
	return namespacedClues{
		mu: &sync.RWMutex{},
		m: map[string]syncValues{
			defaultNamespace: newValues(),
		},
	}
}

func (nc namespacedClues) namespace(name string) syncValues {
	ns, ok := nc.m[name]
	if !ok {
		ns = newValues()
		nc.m[name] = ns
	}

	return ns
}

func (nc namespacedClues) add(name string, toAdd map[string]any) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.namespace(name).add(toAdd)
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type cluesCtxKey struct{}

var key = cluesCtxKey{}

func from(ctx context.Context) namespacedClues {
	nc := ctx.Value(key)

	if nc == nil {
		return newClueMap()
	}

	return nc.(namespacedClues)
}

func set(ctx context.Context, nc namespacedClues) context.Context {
	return context.WithValue(ctx, key, nc)
}

// ---------------------------------------------------------------------------
// data normalization and aggregating
// ---------------------------------------------------------------------------

func normalize(kvs ...any) map[string]any {
	norm := map[string]any{}

	for i := 0; i < len(kvs); i += 2 {
		key := marshal(kvs[i])

		var value any
		if i+1 < len(kvs) {
			value = kvs[i+1]
		}

		norm[key] = value
	}

	return norm
}

func marshal(a any) string {
	if a == nil {
		return ""
	}

	if as, ok := a.(string); ok {
		return as
	}

	if as, ok := a.(fmt.Stringer); ok {
		return as.String()
	}

	bs, err := json.Marshal(a)
	if err != nil {
		return "marshalling clue: " + err.Error()
	}

	return string(bs)
}

// Add adds all key-value pairs to the clues.
func Add(ctx context.Context, kvs ...any) context.Context {
	nc := from(ctx)
	nc.add(defaultNamespace, normalize(kvs...))
	return set(ctx, nc)
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[K comparable, V any](ctx context.Context, m map[K]V) context.Context {
	nc := from(ctx)
	for k, v := range m {
		nc.add(defaultNamespace, normalize(k, v))
	}
	return set(ctx, nc)
}

// AddTo adds all key-value pairs to a namespaced set of clues.
func AddTo(ctx context.Context, namespace string, kvs ...any) context.Context {
	nc := from(ctx)
	nc.add(namespace, normalize(kvs...))
	return set(ctx, nc)
}

// AddMapTo adds a shallow clone of the map to a namespaced set of clues.
func AddMapTo[K comparable, V any](ctx context.Context, namespace string, m map[K]V) context.Context {
	nc := from(ctx)
	for k, v := range m {
		nc.add(namespace, normalize(k, v))
	}
	return set(ctx, nc)
}

// ---------------------------------------------------------------------------
// data retrieval
// ---------------------------------------------------------------------------

// In returns the map of values in the default namespace.
func In(ctx context.Context) values {
	nc := from(ctx)
	return nc.namespace(defaultNamespace).data()
}

// InNamespace returns the map of values in the given namespace.
func InNamespace(ctx context.Context, namespace string) values {
	nc := from(ctx)
	return nc.namespace(namespace).data()
}
