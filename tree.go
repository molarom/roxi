// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0
// Components based on the sort package, Copyright 2010 The Go Authors.

package roxi

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

// edge represents an edge node
type edge struct {
	label byte
	node  *node
}

// edges is a slice of edge nodes
type edges []edge

// get uses binary search to locate a matching node
func (e edges) get(label byte) (*node, bool) {
	l := len(e)

	idx := e.binarySearch(l, label)

	if idx < l && e[idx].label == label {
		return e[idx].node, true
	}
	return nil, false
}

// add inserts an edge into the slice.
//
// The underlying slice is returned to reduce the number
// of heap allocations when inserting an edge.
func (e edges) add(n edge) edges {
	l := len(e)
	idx := e.binarySearch(l, n.label)
	e = append(e, edge{})
	copy(e[idx+1:], e[idx:])
	e[idx] = n
	return e
}

// binarySearch is copied from sort.Search so the function
// call can be inlined.
func (e edges) binarySearch(n int, label byte) int {
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if !(e[h].label >= label) {
			i = h + 1
		} else {
			j = h
		}

	}

	return i
}

// ----------------------------------------------------------------------
// node

// node represents a radix tree node
//
// This tree is just a tailored version of
// gitlab.com/romlaor/radix for http routing.
type node struct {
	key   []byte
	value HandlerFunc
	param bool
	leaf  bool
	edges edges
}

// insert inserts a new key value pair into the tree.
func (n *node) insert(key []byte, value HandlerFunc) {
	insKeyFull := key
	cKeyFull := bytes.NewBuffer(make([]byte, 0, len(key)))

	params := countParams(key)
	current := n
	for len(key) > 0 {
		firstChar := key[0]

		child, ok := current.edges.get(firstChar)
		if !ok {
			// no matching edge, create a new node
			current.edges = current.edges.add(edge{
				label: firstChar,
				node: &node{
					key:   key,
					value: value,
					param: countParams(key) != 0,
					leaf:  true,
				},
			})
			return
		}

		cKeyLen := len(child.key)
		prefixLen := prefixLength(key, child.key)
		if prefixLen == cKeyLen {
			key = key[prefixLen:]
			current = child
			cKeyFull.Write(child.key)
			continue
		}

		// match on param, we've got a conflict.
		if child.param && params != 0 && key[prefixLen-1] == ':' {
			cKeyFull.Write(child.key)
			panic("Only one path variable can be registered per segment: \n" +
				"Route: '" + string(insKeyFull) + "'\n" +
				"Conflicts with: '" + cKeyFull.String() + "'")
		}

		// partial, split and update node
		splitNode := &node{
			key:   child.key[prefixLen:],
			value: child.value,
			param: child.param,
			leaf:  child.leaf,
			edges: child.edges,
		}

		// update child node
		child.key = child.key[:prefixLen]
		child.value = nil
		child.leaf = false
		child.edges = edges{
			edge{
				label: splitNode.key[0],
				node:  splitNode,
			},
		}

		// add node for remainder
		if len(key) > prefixLen {
			child.edges = child.edges.add(edge{
				label: key[prefixLen:][0],
				node: &node{
					key:   key[prefixLen:],
					value: value,
					param: countParams(key[prefixLen:]) != 0,
					leaf:  true,
				},
			})
		} else {
			// no remainder, set value on child
			child.value = value
			child.leaf = true
		}
		return
	}

	pc, file, line, _ := runtime.Caller(2)
	fn := filepath.Base(runtime.FuncForPC(pc).Name())

	panic("Route '" + string(insKeyFull) +
		"' registered in '" +
		fmt.Sprintf("%s() %s:%d", fn, file, line) +
		"' has previously been registered.")
}

// search returns the longest prefix match for a key
func (n *node) search(key []byte, r *http.Request) (HandlerFunc, bool) {
	current := n
	keyLen := len(key)
	for keyLen > 0 {
		firstChar := key[0]
		child, ok := current.edges.get(firstChar)
		if !ok {
			// edge case: check if we're about to match a param
			if child, ok = current.edges.get(':'); !ok {
				// edgier case: check if we're about to match a wildcard
				if child, ok = current.edges.get('*'); !ok {
					break
				}
			}
		}

		// check full match
		cKeyLen := len(child.key)
		if keyLen >= cKeyLen && prefixLength(key[:cKeyLen], child.key) == cKeyLen {
			key = key[cKeyLen:]
			keyLen -= cKeyLen
			current = child
			continue
		}

		// check param match
		if child.param {
			prefixLen := prefixLength(key, child.key)
			lastIdx, ok := parseParams(child.key[prefixLen:], key[prefixLen:], r)
			if !ok {
				// no possible match, early return
				return current.value, false
			}

			current = child
			if len(child.edges) != 0 && lastIdx < keyLen {
				key = key[lastIdx:]
				keyLen -= lastIdx
				continue
			}

		}
		break
	}

	// if we didn't land on a param and the
	// key hasn't been exhausted, it's not a match.
	if !current.param && keyLen != 0 {
		return current.value, false
	}

	return current.value, current.leaf
}

// print recursively prints the tree nodes.
func (n *node) print(level int) {
	if n == nil {
		return
	}

	fmt.Printf("%s[%s]: %v\n", strings.Repeat(" ", level*2), string(n.key), n.value)

	for _, child := range n.edges {
		child.node.print(level + 1)
	}
}

// prefixLength calculates the common prefix length between s1 and s2.
func prefixLength(s1, s2 []byte) int {
	l := len(s1)
	if sz := len(s2); len(s1) > sz {
		l = sz
	}
	length := 0
	for ; length < l && s1[length] == s2[length]; length++ {
	}
	return length
}

// ----------------------------------------------------------------------
// params

// parseParams sets the path value for any registered path variables in b
func parseParams(b []byte, path []byte, r *http.Request) (int, bool) {
	lenB := len(b)
	lenPath := len(path)
	if lenPath == 0 {
		return 0, false
	}

	i, j := 0, 0
	for i < lenB && j < lenPath {

		// Check for param token
		if b[i] == ':' {
			start := i + 1
			for ; b[i] != '/' && i < lenB-1; i++ {
			}
			end := i
			if i == lenB-1 && b[i] != '/' {
				end++
			}

			// grab the path value
			pStart := j
			for ; path[j] != '/' && j < lenPath-1; j++ {
			}
			pEnd := j
			if j == lenPath-1 && path[j] != '/' {
				pEnd++
			}

			r.SetPathValue(toString(b[start:end]), toString(path[pStart:pEnd]))
			continue
		}

		// advance the scan if we're outside of a variable
		if b[i] == path[j] {
			i++
			j++
			continue
		}

		// wildcard is the unlikely case, so check this last.
		if b[i] == '*' {
			start := i + 1
			for ; i < lenB-1; i++ {
			}
			end := i + 1

			// grab the path value
			pStart := j
			for ; j < lenPath-1; j++ {
			}
			pEnd := j + 1

			r.SetPathValue(toString(b[start:end]), toString(path[pStart:pEnd]))
		}

		break
	}

	// if we reached the end for both,
	// or end of b and prev char are eq,
	// it's a match.
	return j, (j >= lenPath-1 && i >= lenB-1) || (path[j-1] == b[lenB-1])
}

// countParams counts the number of params in b.
func countParams(b []byte) (count int) {
	i := 0
	lenB := len(b)
	for i < lenB {

		switch b[i] {
		case ':':
			count++
			for ; b[i] != '/' && i < lenB-1; i++ {
				// Check for bad variable names.
				if b[i+1] == ':' || b[i+1] == '*' {
					panic("path variables cannot contain the following characters: {" +
						"':', '*'" +
						"}\n" +
						"Offending path: '" + string(b) + "'")
				}
			}
		case '*':
			count++
			for ; i < lenB-1; i++ {
				// handle bad wildcard path position
				if b[i+1] == '/' {
					panic("wildcard must be set at the end of the path:\n" +
						"Offending path: '" + string(b) + "'")
				}
			}
		default:
			i++
			continue
		}

		// empty name case.
		if b[i] == ':' || b[i] == '*' {
			panic("path variable has no name: \n" +
				"Offending path: '" + string(b) + "'")
		}
	}

	return
}
