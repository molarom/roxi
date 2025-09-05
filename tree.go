// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0
// Components based on the sort package, Copyright 2010 The Go Authors.

package roxi

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// edge represents an edge node.
type edge struct {
	label byte
	node  *node
}

// edges is a slice of edge nodes.
type edges []edge

// get uses binary search to locate a matching node.
func (e edges) get(label byte) (*node, bool) {
	l := len(e)
	if l > 0 {
		idx := e.binarySearch(l, label)

		if idx < l && e[idx].label == label {
			return e[idx].node, true
		}
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
		h := int(uint(i+j) >> 1) // #nosec G115
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

// node represents a radix tree node.
//
// This tree is just a tailored version of
// gitlab.com/romlaor/radix for http routing.
type node struct {
	key   []byte
	route []byte
	param bool
	leaf  bool
	value HandlerFunc
	edges edges
}

// insert inserts a new key value pair into the tree.
func (n *node) insert(key []byte, value HandlerFunc) {
	// validate params
	params := countParams(key)
	if params != 0 {
		if err := validateParams(key, params); err != nil {
			panic(err)
		}
	}

	insKeyFull := key
	cKeyFull := bytes.NewBuffer(make([]byte, 0, len(key)))

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
					route: insKeyFull,
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

		// mismatch on param, check for conflict.
		if child.param && params != 0 {
			v := (key[prefixLen-1] == ':' && child.key[prefixLen-1] == ':')
			wc := (key[prefixLen-1] == '*' && child.key[prefixLen-1] == '*')

			if v || wc {
				cKeyFull.Write(child.key)
				panic("Only one path variable and wildcard can be registered per path segment: \n" +
					"Route: '" + string(insKeyFull) + "'\n" +
					"Conflicts with: '" + cKeyFull.String() + "'")
			}
		}

		// partial, split and update node
		splitNode := &node{
			key:   child.key[prefixLen:],
			value: child.value,
			route: child.route,
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
					route: insKeyFull,
					value: value,
					param: countParams(key[prefixLen:]) != 0,
					leaf:  true,
				},
			})
		} else {
			// no remainder, set value on child
			child.route = insKeyFull
			child.value = value
			child.leaf = true
		}
		return
	}

	if current.leaf || current.value != nil {
		pc, file, line, _ := runtime.Caller(3)
		fn := filepath.Base(runtime.FuncForPC(pc).Name())

		panic("Route '" + string(insKeyFull) +
			"' registered in '" +
			fmt.Sprintf("%s() %s:%d", fn, file, line) +
			"' has previously been registered.")
	}

	// fix registration bug.
	current.value = value
	current.leaf = true
}

// search returns the longest prefix match for a key.
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

	if r != nil {
		r.Pattern = toString(current.route)
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

// collectRoutes recursively collects all of the routes.
func (n *node) collectRoutes(routes *[]string) {
	if n == nil {
		return
	}

	if n.leaf {
		*routes = append(*routes, string(n.route))
	}

	for _, child := range n.edges {
		child.node.collectRoutes(routes)
	}
}

// prefixLength calculates the common prefix length between s1 and s2.
func prefixLength(s1, s2 []byte) (length int) {
	l := len(s1)
	if sz := len(s2); len(s1) > sz {
		l = sz
	}
	for ; length < l && s1[length] == s2[length]; length++ {
	}
	return length
}

// ----------------------------------------------------------------------
// params

// parseParams sets the path value for any registered path variables in b.
func parseParams(b []byte, path []byte, r *http.Request) (int, bool) {
	lenB := len(b)
	lenPath := len(path)

	if lenPath == 0 {
		if !(checkWildCard(b, 0, lenB) || checkWildCard(b, 1, lenB)) {
			return 0, false
		}
	}

	i, j := 0, 0
	for i < lenB && j < lenPath {

		// Check for param token
		if b[i] == ':' {
			param, end, _ := pathSegment(b, i+1, lenB)
			pValue, pEnd, _ := pathSegment(path, j, lenPath)

			if r != nil {
				r.SetPathValue(toString(param), toString(pValue))
			}

			i, j = end, pEnd

			continue
		}

		// advance the scan if we're outside of a variable
		if b[i] == path[j] {
			i++
			j++
			continue
		}

		break
	}

	// reached the end, early return
	if i == lenB && j == lenPath {
		return j, true
	}

	// wildcard is the unlikely case, so check this last.
	if checkWildCard(b, i, lenB) {
		param, _, _ := pathSegment(b, i+1, lenB)

		// early return for simple lookups
		if r == nil {
			return j, true
		}

		// grab the path value
		if lenPath > 0 {
			r.SetPathValue(toString(param), "/"+toString(path[j:lenPath]))
		} else {
			r.SetPathValue(toString(param), "/")
		}

		if lenPath != 0 {
			return lenPath - 1, true
		}
		return 0, true
	}

	// not at the end, return if chars are different.
	if i == 0 || j == 0 {
		return 0, (path[0] == b[0] && lenPath-1 != 0)
	}
	return j, (path[j-1] == b[i-1] && lenPath-1 != j-1)
}

func checkWildCard(b []byte, idx, l int) bool {
	if idx >= l {
		return false
	}
	if b[idx] != '*' {
		return false
	}
	return true
}

func countParams(b []byte) (count int) {
	lenB := len(b)
	for i := 0; i < lenB; i++ {
		switch b[i] {
		case ':', '*':
			count++
		}
	}
	return count
}

func pathSegment(b []byte, start, length int) ([]byte, int, bool) {
	if length == 0 || start >= length {
		return nil, -1, false
	}

	end := start
	for ; end < length; end++ {
		switch b[end] {
		case '/':
			return b[start:end], end, true
		case '*', ':':
			return nil, -1, false
		}
	}

	return b[start:end], end, true
}

func validateParams(b []byte, total int) error {
	i, count := 0, 0
	lenB := len(b)
	for ; i < lenB; i++ {
		switch b[i] {
		case ':':
			param, _, valid := pathSegment(b, i+1, lenB)
			if !valid {
				return errors.New("path variables cannot contain the following characters: {" +
					"':', '*'" +
					"}\n" +
					"path: '" + string(b) + "' is not valid.")
			}
			if len(param) == 0 {
				return errors.New("missing name for variable in path: \n" +
					"'" + string(b) + "'")
			}
			count++
		case '*':
			param, end, valid := pathSegment(b, i+1, lenB)
			if !valid {
				return errors.New("path variables cannot contain the following characters: {" +
					"':', '*'" +
					"}\n" +
					"path: '" + string(b) + "' is not valid.")
			}
			if len(param) == 0 {
				return errors.New("missing name for variable in path: \n" +
					"'" + string(b) + "'")
			}
			if end != lenB {
				return errors.New("wildcard must be set at the end of the path:\n" +
					"path: '" + string(b) + "' is not valid.")
			}
			count++
		}
	}
	// sanity check
	if count != total {
		return errors.New("variable count mismatch for path:\n'" +
			string(b) +
			"'; got[" + strconv.Itoa(count) + "]; want[" + strconv.Itoa(total) + "]")
	}
	return nil
}
