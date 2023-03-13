package clues

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/exp/maps"
)

// ---------------------------------------------------------------------------
// structure data storage and namespaces
// ---------------------------------------------------------------------------

const defaultNamespace = "clue_ns_default"

type values map[string]any

func (vs values) add(m map[string]any) values {
	v2 := maps.Clone(vs)
	if v2 == nil {
		v2 = values{}
	}

	maps.Copy(v2, m)
	return v2
}

func (vs values) Slice() []any {
	s := make([]any, 0, 2*len(vs))

	for k, v := range vs {
		s = append(s, k, v)
	}

	return s
}

// outer map tracks namespaces
// inner map tracks key/value pairs
type namespacedClues map[string]values

func newClueMap() namespacedClues {
	return namespacedClues{defaultNamespace: values{}}
}

func (nc namespacedClues) namespace(name string) values {
	ns, ok := nc[name]
	if !ok {
		nc[name] = values{}
	}

	return ns
}

func (nc namespacedClues) add(name string, toAdd map[string]any) namespacedClues {
	nc2 := maps.Clone(nc)
	if nc2 == nil {
		nc2 = newClueMap()
	}

	vs := nc2.namespace(name)
	nc2[name] = vs.add(toAdd)

	return nc2
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
			value = marshal(kvs[i+1])
		}

		norm[key] = value
	}

	return norm
}

func marshal(a any) string {
	if a == nil {
		return ""
	}

	if as, ok := a.(Concealer); ok {
		return as.Conceal()
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
	return set(ctx, nc.add(defaultNamespace, normalize(kvs...)))
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[K comparable, V any](ctx context.Context, m map[K]V) context.Context {
	nc := from(ctx)

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return set(ctx, nc.add(defaultNamespace, normalize(kvs...)))
}

// AddTo adds all key-value pairs to a namespaced set of clues.
func AddTo(ctx context.Context, namespace string, kvs ...any) context.Context {
	nc := from(ctx)
	return set(ctx, nc.add(namespace, normalize(kvs...)))
}

// AddMapTo adds a shallow clone of the map to a namespaced set of clues.
func AddMapTo[K comparable, V any](ctx context.Context, namespace string, m map[K]V) context.Context {
	nc := from(ctx)

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return set(ctx, nc.add(namespace, normalize(kvs...)))
}

// ---------------------------------------------------------------------------
// data retrieval
// ---------------------------------------------------------------------------

// In returns the map of values in the default namespace.
func In(ctx context.Context) values {
	nc := from(ctx)
	return nc.namespace(defaultNamespace)
}

// InNamespace returns the map of values in the given namespace.
func InNamespace(ctx context.Context, namespace string) values {
	nc := from(ctx)
	return nc.namespace(namespace)
}
