package clues

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Adder interface {
	Add(key string, n int64)
}

// ---------------------------------------------------------------------------
// structure data storage and namespaces
// ---------------------------------------------------------------------------

// dataNodes contain the data tracked by both clues in contexts and in errors.
//
// These nodes create an inverted tree, such that nodes can walk their ancestry
// path from leaf (the current node) to root (the highest ancestor), but not
// from root to child.  This allows clues to establish sets of common ancestor
// data with unique branches for individual descendants, making the addition of
// new data inherently theadsafe.
//
// For collisions during aggregation, distance from the root denotes priority,
// with the root having the lowest priority.  IE: if a child overwrites a key
// declared by an ancestor, the child's entry takes priority.
type dataNode struct {
	parent       *dataNode
	id           string
	vs           map[string]any
	labelCounter Adder
	comment      comment
}

func makeNodeID() string {
	uns := uuid.NewString()
	return uns[:4] + uns[len(uns)-4:]
}

func (dn *dataNode) add(m map[string]any) *dataNode {
	if m == nil {
		m = map[string]any{}
	}

	return &dataNode{
		parent:       dn,
		id:           makeNodeID(),
		vs:           maps.Clone(m),
		labelCounter: dn.labelCounter,
	}
}

func (dn *dataNode) trace(name string) *dataNode {
	if name == "" {
		name = makeNodeID()
	}

	return &dataNode{
		parent:       dn,
		id:           name,
		vs:           map[string]any{},
		labelCounter: dn.labelCounter,
	}
}

// lineage runs the fn on every valueNode in the ancestry tree,
// starting at the root and ending at the dataNode.
func (dn *dataNode) lineage(fn func(id string, vs map[string]any)) {
	if dn == nil {
		return
	}

	if dn.parent != nil {
		dn.parent.lineage(fn)
	}

	fn(dn.id, dn.vs)
}

func (dn *dataNode) Slice() []any {
	m := dn.Map()
	s := make([]any, 0, 2*len(m))

	for k, v := range m {
		s = append(s, k, v)
	}

	return s
}

func (dn *dataNode) Map() map[string]any {
	var (
		m    = map[string]any{}
		idsl = []string{}
	)

	dn.lineage(func(id string, vs map[string]any) {
		if len(id) > 0 {
			idsl = append(idsl, id)
		}

		for k, v := range vs {
			m[k] = v
		}
	})

	if len(idsl) > 0 {
		m["clues_trace"] = strings.Join(idsl, ",")
	}

	return m
}

type comment struct {
	Caller  string
	Dir     string
	File    string
	Message string
}

func (c comment) isEmpty() bool {
	return len(c.Message) == 0
}

func newComment(
	depth int,
	template string,
	values ...any,
) comment {
	caller := getCaller(depth + 1)
	longTrace := getTrace(depth + 1)
	dir, file := path.Split(longTrace)

	return comment{
		Caller:  caller,
		Dir:     dir,
		File:    file,
		Message: fmt.Sprintf(template, values...),
	}
}

func (dn *dataNode) addComment(
	depth int,
	msg string,
	vs ...any,
) *dataNode {
	if len(msg) == 0 {
		return dn
	}

	return &dataNode{
		parent:       dn,
		labelCounter: dn.labelCounter,
		comment:      newComment(depth+1, msg, vs...),
	}
}

type comments []comment

func (cs comments) String() string {
	result := []string{}

	for _, c := range cs {
		result = append(result, c.Caller+" - "+c.File)
		result = append(result, "\t"+c.Message)
	}

	return strings.Join(result, "\n")
}

// Comments retrieves the full ancestor comment chain.
// The return value is ordered from the first added comment
// to the last one.
func (dn *dataNode) Comments() comments {
	result := comments{}

	if !dn.comment.isEmpty() {
		result = append(result, dn.comment)
	}

	for dn.parent != nil {
		dn = dn.parent
		if !dn.comment.isEmpty() {
			result = append(result, dn.comment)
		}
	}

	slices.Reverse(result)

	return result
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type cluesCtxKey string

const defaultNamespace cluesCtxKey = "default_clues_namespace_key"

func ctxKey(namespace string) cluesCtxKey {
	return cluesCtxKey(namespace)
}

func from(ctx context.Context, namespace cluesCtxKey) *dataNode {
	dn := ctx.Value(namespace)

	if dn == nil {
		return &dataNode{}
	}

	return dn.(*dataNode)
}

func set(ctx context.Context, dn *dataNode) context.Context {
	return context.WithValue(ctx, defaultNamespace, dn)
}

func setTo(ctx context.Context, namespace string, dn *dataNode) context.Context {
	return context.WithValue(ctx, ctxKey(namespace), dn)
}

// ---------------------------------------------------------------------------
// data normalization and aggregating
// ---------------------------------------------------------------------------

func normalize(kvs ...any) map[string]any {
	norm := map[string]any{}

	for i := 0; i < len(kvs); i += 2 {
		key := marshal(kvs[i], true)

		var value any
		if i+1 < len(kvs) {
			value = marshal(kvs[i+1], true)
		}

		norm[key] = value
	}

	return norm
}

func marshal(a any, conceal bool) string {
	if a == nil {
		return ""
	}

	// protect against nil pointer values with value-receiver funcs
	rvo := reflect.ValueOf(a)
	if rvo.Kind() == reflect.Ptr && rvo.IsNil() {
		return ""
	}

	if as, ok := a.(Concealer); conceal && ok {
		return as.Conceal()
	}

	if as, ok := a.(string); ok {
		return as
	}

	if as, ok := a.(fmt.Stringer); ok {
		return as.String()
	}

	return fmt.Sprintf("%+v", a)
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

// AddTrace stacks a clues node onto this context.  Adding a node ensures
// that this point in code is identified by an ID, which can later be
// used to correlate and isolate logs to certain trace branches.
// AddTrace is only needed for layers that don't otherwise call Add() or
// similar functions, since those funcs already attach a new node.
func AddTrace(ctx context.Context) context.Context {
	nc := from(ctx, defaultNamespace)
	return set(ctx, nc.trace(""))
}

// AddTraceTo stacks a clues node onto this context within the specified
// namespace.  Adding a node ensures that a point in code is identified
// by an ID, which can later be used to correlate and isolate logs to
// certain trace branches.  AddTraceTo is only needed for layers that don't
// otherwise call AddTo() or similar functions, since those funcs already
// attach a new node.
func AddTraceTo(ctx context.Context, namespace string) context.Context {
	nc := from(ctx, ctxKey(namespace))
	return setTo(ctx, namespace, nc.trace(""))
}

// AddTraceName stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddTraceName
// can be called without additional values if you only want to add a trace marker.
func AddTraceName(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	nc := from(ctx, defaultNamespace)

	var node *dataNode
	if len(kvs) > 0 {
		node = nc.add(normalize(kvs...))
		node.id = name
	} else {
		node = nc.trace(name)
	}

	return set(ctx, node)
}

// AddTraceNameTo stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddTraceNameTo
// can be called without additional values if you only want to add a trace marker.
func AddTraceNameTo(
	ctx context.Context,
	name, namespace string,
	kvs ...any,
) context.Context {
	nc := from(ctx, ctxKey(namespace))

	var node *dataNode
	if len(kvs) > 0 {
		node = nc.add(normalize(kvs...))
		node.id = name
	} else {
		node = nc.trace(name)
	}

	return setTo(ctx, namespace, node)
}

// AddLabelCounter embeds an Adder interface into this context. Any already
// embedded Adder will get replaced.  When adding Labels to a clues.Err the
// LabelCounter will use the label as the key for the Add call, and increment
// the count of that label by one.
func AddLabelCounter(ctx context.Context, counter Adder) context.Context {
	nc := from(ctx, defaultNamespace)
	nn := nc.add(nil)
	nn.labelCounter = counter
	return set(ctx, nn)
}

// Comments are special case additions to the context.  They're here to, well,
// let you add comments!  Why?  Because sometimes it's not sufficient to have a
// log let you know that a line of code was reached. Even a bunch of clues to
// describe system state may not be enough.  Sometimes what you need in order
// to debug the situation is a long-form explanation (you do already add that
// to your code, don't you?).  Or, even better, a linear history of long-form
// explanations, each one building on the prior (which you can't easily do in
// code).
//
// Should you transfer all your comments to clues?  Absolutely not.  But in
// cases where extra explantion is truly important to debugging production,
// when all you've got are some logs and (maybe if you're lucky) a span trace?
// Those are the ones you want.
//
// Unlike other additions, which are added as top-level key:value pairs to the
// context, comments are all held as a single array of additions, persisted in
// order of appearance, and prefixed by the file and line in which they appeared.
// This means comments are always added to the context and never clobber each
// other, regardless of their location.  IE: don't add them to a loop.
func AddComment(ctx context.Context, msg string, vs ...any) context.Context {
	nc := from(ctx, defaultNamespace)
	nn := nc.addComment(1, msg, vs...)

	return set(ctx, nn)
}

// ---------------------------------------------------------------------------
// data retrieval
// ---------------------------------------------------------------------------

// In returns the map of values in the default namespace.
func In(ctx context.Context) *dataNode {
	return from(ctx, defaultNamespace)
}

// InNamespace returns the map of values in the given namespace.
func InNamespace(ctx context.Context, namespace string) *dataNode {
	return from(ctx, ctxKey(namespace))
}
