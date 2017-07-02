package object

import (
	"fmt"
)

// MergeError is returned by Merge function on conflict
// during merging two object contents.
type MergeError struct {
	Path []string    // object path starting at object root of conflicting element
	In   interface{} // conflicting element in user's object
	Out  interface{} // conflicting element in kloud's object
}

// Error implements the builtin error interface.
func (me *MergeError) Error() string {
	return fmt.Sprintf("unable to merge incompatible values for %v key: in=%T, out=%T", me.Path, me.In, me.Out)
}

// Merge merges two objects into one.
//
// If during merge any conflict is encountered, Merge fails with non-nil
// error that is of *MergeError type, which describes details of the conflict.
func Merge(in, out map[string]interface{}) error {
	type iter struct {
		path []string
		in   map[string]interface{}
		out  map[string]interface{}
	}

	i, stack := iter{}, []iter{{nil, in, out}}

	for len(stack) != 0 {
		i, stack = stack[0], stack[1:]

		for k, v := range i.in {
			switch in := v.(type) {
			case map[string]interface{}:
				switch out := i.out[k].(type) {
				case nil:
					i.out[k] = in
				case map[string]interface{}:
					stack = append(stack, iter{
						path: append(i.path, k),
						in:   in,
						out:  out,
					})
				default:
					return &MergeError{
						Path: append(i.path, k),
						In:   in,
						Out:  out,
					}
				}
			case []interface{}:
				switch out := i.out[k].(type) {
				case nil:
				case []interface{}:
					for _, elm := range in {
						out = append(out, elm)
					}

					in = out
				default:
					return &MergeError{
						Path: append(i.path, k),
						In:   in,
						Out:  out,
					}
				}

				i.out[k] = in
			default:
				i.out[k] = in
			}
		}
	}

	return nil
}
