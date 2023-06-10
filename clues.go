package clues

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"golang.org/x/exp/maps"
)

// ---------------------------------------------------------------------------
// structure data storage and namespaces
// ---------------------------------------------------------------------------

type valueNode struct {
	parent *valueNode
	vs     map[string]any
}

func newNode(m map[string]any) *valueNode {
	return &valueNode{vs: m}
}

func (vn *valueNode) add(m map[string]any) *valueNode {
	return &valueNode{
		parent: vn,
		vs:     maps.Clone(m),
	}
}

// lineage runs the fn on every valueNode in the ancestry tree,
// starting at the root and ending at vn.
func (vn *valueNode) lineage(fn func(vs map[string]any)) {
	if vn == nil {
		return
	}

	if vn.parent != nil {
		vn.parent.lineage(fn)
	}

	fn(vn.vs)
}

func (vn *valueNode) Slice() []any {
	m := vn.Map()
	s := make([]any, 0, 2*len(m))

	for k, v := range m {
		s = append(s, k, v)
	}

	return s
}

func (vn *valueNode) Map() map[string]any {
	m := map[string]any{}

	vn.lineage(func(vs map[string]any) {
		for k, v := range vs {
			m[k] = v
		}
	})

	return m
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type cluesCtxKey string

const defaultNamespace cluesCtxKey = "default_clues_namespace_key"

func ctxKey(namespace string) cluesCtxKey {
	return cluesCtxKey(namespace)
}

func from(ctx context.Context, namespace cluesCtxKey) *valueNode {
	vn := ctx.Value(namespace)

	if vn == nil {
		return &valueNode{}
	}

	return vn.(*valueNode)
}

func set(ctx context.Context, vn *valueNode) context.Context {
	return context.WithValue(ctx, defaultNamespace, vn)
}

func setTo(ctx context.Context, namespace string, vn *valueNode) context.Context {
	return context.WithValue(ctx, ctxKey(namespace), vn)
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

	// protect against nil pointer values with value-receiver funcs
	rvo := reflect.ValueOf(a)
	if rvo.Kind() == reflect.Ptr && rvo.IsNil() {
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
	nc := from(ctx, defaultNamespace)
	return set(ctx, nc.add(normalize(kvs...)))
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[K comparable, V any](ctx context.Context, m map[K]V) context.Context {
	nc := from(ctx, defaultNamespace)

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return set(ctx, nc.add(normalize(kvs...)))
}

// AddTo adds all key-value pairs to a namespaced set of clues.
func AddTo(ctx context.Context, namespace string, kvs ...any) context.Context {
	nc := from(ctx, ctxKey(namespace))
	return setTo(ctx, namespace, nc.add(normalize(kvs...)))
}

// AddMapTo adds a shallow clone of the map to a namespaced set of clues.
func AddMapTo[K comparable, V any](ctx context.Context, namespace string, m map[K]V) context.Context {
	nc := from(ctx, ctxKey(namespace))

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return setTo(ctx, namespace, nc.add(normalize(kvs...)))
}

// ---------------------------------------------------------------------------
// data retrieval
// ---------------------------------------------------------------------------

// In returns the map of values in the default namespace.
func In(ctx context.Context) *valueNode {
	return from(ctx, defaultNamespace)
}

// InNamespace returns the map of values in the given namespace.
func InNamespace(ctx context.Context, namespace string) *valueNode {
	return from(ctx, ctxKey(namespace))
}
