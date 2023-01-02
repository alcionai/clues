package clues

import (
	"context"
	"encoding/json"
	"fmt"
)

const defaultNamespace = "clue_ns_default"

// outer map tracks namespaces
// inner map tracks key/value pairs
type namespacedClues map[string]map[string]any

func newClueMap() namespacedClues {
	return namespacedClues{defaultNamespace: map[string]any{}}
}

func (nc namespacedClues) namespace(name string) map[string]any {
	ns, ok := nc[name]
	if !ok {
		ns = map[string]any{}
		nc[name] = ns
	}

	return ns
}

func (nc namespacedClues) add(name, key string, value any) {
	ns := nc.namespace(name)
	ns[key] = value
}

func (nc namespacedClues) addAll(name string, kvs ...any) {
	for i := 0; i < len(kvs); i += 2 {
		key := marshal(kvs[i])

		var value any
		if i+1 < len(kvs) {
			value = kvs[i+1]
		}

		nc.add(defaultNamespace, key, value)
	}
}

type cluesCtxKey struct{}

func from(ctx context.Context) namespacedClues {
	am := ctx.Value(cluesCtxKey{})

	if am == nil {
		return newClueMap()
	}

	return am.(namespacedClues)
}

func set(ctx context.Context, nc namespacedClues) context.Context {
	return context.WithValue(ctx, cluesCtxKey{}, nc)
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

// Add a key-value pair to the clues.
func Add(ctx context.Context, key string, value any) context.Context {
	nc := from(ctx)
	nc.add(defaultNamespace, key, value)
	return set(ctx, nc)
}

// AddAll adds all key-value pairs to the clues.
func AddAll(ctx context.Context, kvs ...any) context.Context {
	nc := from(ctx)
	nc.addAll(defaultNamespace, kvs...)
	return set(ctx, nc)
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMapT[K comparable, V any](ctx context.Context, m map[K]V) context.Context {
	nc := from(ctx)
	for k, v := range m {
		nc.add(defaultNamespace, marshal(k), v)
	}
	return set(ctx, nc)
}

// Add a key-value pair to a namespaced set of clues.
func AddTo(ctx context.Context, namespace, key string, value any) context.Context {
	nc := from(ctx)
	nc.add(namespace, key, value)
	return set(ctx, nc)
}

// AddAllTo adds all key-value pairs to a namespaced set of clues.
func AddAllTo(ctx context.Context, namespace string, kvs ...any) context.Context {
	nc := from(ctx)
	nc.addAll(namespace, kvs...)
	return set(ctx, nc)
}

// AddMapTo adds a shallow clone of the map to a namespaced set of clues.
func AddMapTo[K comparable, V any](ctx context.Context, namespace string, m map[K]V) context.Context {
	nc := from(ctx)
	for k, v := range m {
		nc.add(namespace, marshal(k), v)
	}
	return set(ctx, nc)
}

// Values returns the map of values in the default namespace.
func Values(ctx context.Context) map[string]any {
	nc := from(ctx)
	return nc.namespace(defaultNamespace)
}

// Namespace returns the map of values in the given namespace.
func Namespace(ctx context.Context, namespace string) map[string]any {
	nc := from(ctx)
	return nc.namespace(namespace)
}

func asSlice(ns map[string]any) []any {
	s := make([]any, 0, 2*len(ns))

	for k, v := range ns {
		s = append(s, k, v)
	}

	return s
}

// Slice returns all the values in the default namespace as a slice
// of alternating key, value pairs.
func Slice(ctx context.Context) []any {
	ns := Values(ctx)
	return asSlice(ns)
}

// NameSlice returns all the values in the given namespace as a slice
// of alternating key, value pairs.
func NameSlice(ctx context.Context, namespace string) []any {
	ns := Namespace(ctx, namespace)
	return asSlice(ns)
}
