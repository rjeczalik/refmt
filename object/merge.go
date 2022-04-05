package object

import (
	"fmt"
	"sort"
	"strings"
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

func Flatten(in map[string]interface{}, sep string) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}

	root := make(map[string]interface{})

	type elm struct {
		parent map[string]interface{}
		key    []string
		left   []string
	}

	var (
		it    elm
		k     string
		queue = []elm{{parent: in, left: Keys(in)}}
	)

	for len(queue) != 0 {
		it, queue = queue[len(queue)-1], queue[:len(queue)-1]
		k, it.left = it.left[0], it.left[1:]

		key := clone(it.key, k)

		if len(it.left) != 0 {
			queue = append(queue, it)
		}

		switch v := it.parent[k].(type) {
		case []interface{}:
			m := make(map[string]interface{}, len(v))

			for i, v := range v {
				m[fmt.Sprint(i)] = v
			}

			queue = append(queue, elm{
				parent: m,
				key:    key,
				left:   Keys(m),
			})
		case map[string]interface{}:
			queue = append(queue, elm{
				parent: v,
				key:    key,
				left:   Keys(v),
			})
		default:
			if len(key) != 0 {
				root[strings.Join(key, sep)] = v
			}
		}
	}

	return root
}

func SetFlatKeyValue(m map[string]interface{}, key, value string) error {
	keys := strings.Split(key, ".")
	it := m
	last := len(keys) - 1

	for _, key := range keys[:last] {
		switch v := it[key].(type) {
		case map[string]interface{}:
			it = v
		case nil:
			newV := make(map[string]interface{})
			it[key] = newV
			it = newV
		default:
			return fmt.Errorf("key is not an object")
		}
	}

	if value == "" {
		delete(it, keys[last])
	} else {
		it[keys[last]] = value
	}

	return nil
}

func clone(s []string, vs ...string) []string {
	sCopy := make([]string, len(s), len(s)+len(vs))
	copy(sCopy, s)

	for _, v := range vs {
		if v != "" && v != "-" {
			sCopy = append(sCopy, v)
		}
	}

	return sCopy
}

func Keys(in map[string]interface{}) []string {
	keys := make([]string, 0, len(in))

	for k := range in {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
