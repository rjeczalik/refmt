package object

import "fmt"

func Walk(m map[string]interface{}, fn func(map[string]interface{}, string)) {
	type elm struct {
		parent map[string]interface{}
		key    []string
		left   []string
	}

	var (
		it    elm
		k     string
		queue = []elm{{parent: m, left: Keys(m)}}
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
				fn(it.parent, k)
			}
		}
	}
}
