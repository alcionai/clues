package node

import (
	"fmt"
	"slices"
	"strings"
)

// ---------------------------------------------------------------------------
// comments
// ---------------------------------------------------------------------------

type Comment struct {
	// the func name in which the comment was created.
	Caller string
	// the name of the file owning the caller.
	File string
	// the comment message itself.
	Message string
}

// shorthand for checking if an empty comment was generated.
func (c Comment) IsEmpty() bool {
	return len(c.Message) == 0
}

// NewComment formats the provided values, and grabs the caller and trace
// info according to the depth.  Depth is a skip-caller count, and any func
// calling this one should provide either `1` (for itself) or `depth+1` (if
// it was already given a depth value).
func NewComment(
	depth int,
	template string,
	values ...any,
) Comment {
	caller := GetCaller(depth + 1)
	_, _, parentFileLine := GetDirAndFile(depth + 1)

	return Comment{
		Caller:  caller,
		File:    parentFileLine,
		Message: fmt.Sprintf(template, values...),
	}
}

// AddComment creates a new nodewith a comment but no other properties.
func (dn *Node) AddComment(
	depth int,
	msg string,
	vs ...any,
) *Node {
	if len(msg) == 0 {
		return dn
	}

	spawn := dn.SpawnDescendant()
	spawn.ID = randomNodeID()
	spawn.Comment = NewComment(depth+1, msg, vs...)

	return spawn
}

// CommentHistory allows us to put a stringer on a slice of CommentHistory.
type CommentHistory []Comment

// String formats the slice of comments as a stack, much like you'd see
// with an error stacktrace.  Comments are listed top-to-bottom from first-
// to-last.
//
// The format for each comment in the stack is:
//
//	<caller> - <file>:<line>
//	  <message>
func (cs CommentHistory) String() string {
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
func (dn *Node) Comments() CommentHistory {
	result := CommentHistory{}

	if !dn.Comment.IsEmpty() {
		result = append(result, dn.Comment)
	}

	for dn.Parent != nil {
		dn = dn.Parent
		if !dn.Comment.IsEmpty() {
			result = append(result, dn.Comment)
		}
	}

	slices.Reverse(result)

	return result
}
