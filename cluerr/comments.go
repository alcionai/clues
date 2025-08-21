package cluerr

import (
	"github.com/alcionai/clues/internal/errs"
	"github.com/alcionai/clues/internal/node"
)

// ------------------------------------------------------------
// comments
// ------------------------------------------------------------

// Comments retrieves all comments in the error.
func (err *Err) Comments() node.CommentHistory {
	return Comments(err)
}

// Comments retrieves all comments in the error.
func Comments(err error) node.CommentHistory {
	if errs.IsNilIface(err) {
		return node.CommentHistory{}
	}

	ancs := ancestors(err)
	result := node.CommentHistory{}

	for _, ancestor := range ancs {
		ce, ok := ancestor.(*Err)
		if !ok {
			continue
		}

		result = append(result, ce.data.Comments()...)
	}

	return result
}

// Comment is a special case additions to the error.  They're here to, well,
// let you add comments!  Why?  Because sometimes it's not sufficient to have
// an error message describe what that error really means. Even a bunch of
// clues  to describe system state may not be enough.  Sometimes what you need
// in order to debug the situation is a long-form explanation (you do already
// add that to your code, don't you?).  Or, even better, a linear history of
// long-form explanations, each one building on the prior (which you can't
// easily do in code).
//
// Unlike other additions, which are added as top-level key:value pairs to the
// context, the whole history of comments gets retained, persisted in order of
// appearance and prefixed by the file and line in which they appeared. This
// means comments are always added to the error and never clobber each other,
// regardless of their location.
func (err *Err) Comment(msg string, vs ...any) *Err {
	if errs.IsNilIface(err) {
		return nil
	}

	return &Err{
		e: err,
		// have to do a new node here, or else comments will duplicate
		data: &node.Node{Comment: node.NewComment(1, msg, vs...)},
	}
}
