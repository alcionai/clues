package clues

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"slices"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

// ---------------------------------------------------------------------------
// data nodes
// ---------------------------------------------------------------------------

type Adder interface {
	Add(key string, n int64)
}

// dataNodes contain the data tracked by both clues in contexts and in errors.
//
// These nodes compose a tree, such that nodes can walk their ancestry path from
// leaf (the current node) to root (the highest ancestor), but not from root to
// child.  This allows clues to establish sets of common ancestor data with unique
// branches for individual descendants, making the addition of new data inherently
// theadsafe.
//
// For collisions during aggregation, distance from the root denotes priority,
// with the root having the lowest priority.  IE: if a child overwrites a key
// declared by an ancestor, the child's entry takes priority.
type dataNode struct {
	parent *dataNode

	// ids are optional and are used primarily as tracing markers.
	// if empty, the trace for that node will get skipped when building the
	// full trace along the node's ancestry path in the tree.
	id string

	// values are they arbitrary key:value pairs that appear in clues when callers
	// use the Add(ctx, k, v) or err.With(k, v) adders.  Each key-value pair added
	// to the node is used to produce the final set of Values() in the dataNode,
	// with lower nodes in the tree taking priority over higher nodes for any
	// collision resolution.
	values map[string]any

	// each node can hold a single commment.  The history of comments produced
	// by the ancestry path through the tree will get concatenated from oldest
	// ancestor to the current node to produce the comment history.
	comment comment

	// labelCounter is a func hook that allows a caller to automatically count the
	// number of times a label appears.  DataNodes themselves have no labels, so
	// in this case the presence of a labelCounter will be used to count the labels
	// appearing in errors which attach this data node to the error.
	//
	// Errors will only utilize the first labelCounter they find.  The tree is searched
	// from leaf to root when looking for populated labelCounters.
	labelCounter Adder
}

// ---------------------------------------------------------------------------
// setters
// ---------------------------------------------------------------------------

// normalize ensures that the variadic of key-value pairs is even in length,
// and then transforms that slice of values into a map[string]any, where all
// keys are transformed to string using the marshal() func.
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

// marshal is the central marshalling handler for the entire package.  All
// stringification of values comes down to this function.  Priority for
// stringification follows this order:
// 1. nil -> ""
// 2. conceal all concealer interfaces
// 3. flat string values
// 4. string all stringer interfaces
// 5. fmt.sprintf the rest
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

// addValues adds all entries in the map to the dataNode's values.
func (dn *dataNode) addValues(m map[string]any) *dataNode {
	if m == nil {
		m = map[string]any{}
	}

	return &dataNode{
		parent:       dn,
		id:           makeNodeID(),
		values:       maps.Clone(m),
		labelCounter: dn.labelCounter,
	}
}

// trace adds a new leaf containing a trace ID and no other values.
func (dn *dataNode) trace(name string) *dataNode {
	if name == "" {
		name = makeNodeID()
	}

	return &dataNode{
		parent:       dn,
		id:           name,
		values:       map[string]any{},
		labelCounter: dn.labelCounter,
	}
}

// ---------------------------------------------------------------------------
// getters
// ---------------------------------------------------------------------------

// lineage runs the fn on every valueNode in the ancestry tree,
// starting at the root and ending at the dataNode.
func (dn *dataNode) lineage(fn func(id string, vs map[string]any)) {
	if dn == nil {
		return
	}

	if dn.parent != nil {
		dn.parent.lineage(fn)
	}

	fn(dn.id, dn.values)
}

// In returns the default dataNode from the context.
// TODO: turn return an interface instead of a dataNode, have dataNodes
// and errors both comply with that wrapper.
func In(ctx context.Context) *dataNode {
	return nodeFromCtx(ctx, defaultNamespace)
}

// InNamespace returns the map of values in the given namespace.
// TODO: turn return an interface instead of a dataNode, have dataNodes
// and errors both comply with that wrapper.
func InNamespace(ctx context.Context, namespace string) *dataNode {
	return nodeFromCtx(ctx, ctxKey(namespace))
}

// Map flattens the tree of dataNode.values into a map.  Descendant nodes
// take priority over ancestors in cases of collision.
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

// Slice flattens the tree of dataNode.values into a Slice where all even
// indices contain the keys, and all odd indices contain values.  Descendant
// nodes take priority over ancestors in cases of collision.
func (dn *dataNode) Slice() []any {
	m := dn.Map()
	s := make([]any, 0, 2*len(m))

	for k, v := range m {
		s = append(s, k, v)
	}

	return s
}

// ---------------------------------------------------------------------------
// comments
// ---------------------------------------------------------------------------

type comment struct {
	// the func name in which the comment was created.
	Caller string
	// the directory path of the file owning the Caller.
	Dir string
	// the name of the file owning the caller.
	File string
	// the comment message itself.
	Message string
}

// shorthand for checking if an empty comment was generated.
func (c comment) isEmpty() bool {
	return len(c.Message) == 0
}

// newComment formats the provided values, and grabs the caller and trace
// info according to the depth.  Depth is a skip-caller count, and any func
// calling this one should provide either `1` (for itself) or `depth+1` (if
// it was already given a depth value).
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

// addComment creates a new dataNode with a comment but no other properties.
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

// comments allows us to put a stringer on a slice of comments.
type comments []comment

// String formats the slice of comments as a stack, much like you'd see
// with an error stacktrace.  Comments are listed top-to-bottom from first-
// to-last.
//
// The format for each comment in the stack is:
//
//	<caller> - <file>:<line>
//	  <message>
func (cs comments) String() string {
	result := []string{}

	for _, c := range cs {
		result = append(result, c.Caller+" - "+c.File)
		result = append(result, "\t"+c.Message)
	}

	return strings.Join(result, "\n")
}

// Comments retrieves the full ancestor comment chain.
// The return value is ordered from the first added comment (closest to
// the root) to the most recent one (closest to the leaf).
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

// nodeFromCtx pulls the datanode within a given namespace out of the context.
func nodeFromCtx(
	ctx context.Context,
	namespace cluesCtxKey,
) *dataNode {
	dn := ctx.Value(namespace)

	if dn == nil {
		return &dataNode{}
	}

	return dn.(*dataNode)
}

// setDefaultNodeInCtx adds the context to the dataNode within the given
// namespace and returns the updated context.
func setDefaultNodeInCtx(
	ctx context.Context,
	dn *dataNode,
) context.Context {
	return context.WithValue(ctx, defaultNamespace, dn)
}

// setNodeInCtx adds the context to the dataNode within the given namespace
// and returns the updated context.
func setNodeInCtx(
	ctx context.Context,
	namespace string,
	dn *dataNode,
) context.Context {
	return context.WithValue(ctx, ctxKey(namespace), dn)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// makeNodeID generates a random hash of 8 characters for use as a node ID.
func makeNodeID() string {
	uns := uuid.NewString()
	return uns[:4] + uns[len(uns)-4:]
}

// getTrace retrieves the file and line number of the caller. Depth is the
// skip-caller count.  Clues funcs that call this one should provide either
// `1` (if they do not already have a depth value), or `depth+1` otherwise`.
//
// Formats to: `<file>:<line>`
func getTrace(depth int) string {
	_, file, line, _ := runtime.Caller(depth + 1)
	return fmt.Sprintf("%s:%d", file, line)
}

// getCaller retrieves the func name of the caller. Depth is the  skip-caller
// count.  Clues funcs that call this one should provide either `1` (if they
// do not already have a depth value), or `depth+1` otherwise.`
func getCaller(depth int) string {
	pc, _, _, ok := runtime.Caller(depth + 1)
	if !ok {
		return ""
	}

	funcPath := runtime.FuncForPC(pc).Name()
	base := path.Base(funcPath)
	parts := strings.Split(base, ".")

	if len(parts) < 2 {
		return base
	}

	return parts[len(parts)-1]
}
