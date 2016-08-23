package memory

import (
	"errors"
	"reflect"
	"sort"

	stream "github.com/ipfs/go-ipld/coding/stream"
	mh "github.com/jbenet/go-multihash"
)

// These are the constants used in the format.
const (
	LinkKey = "/" // key for merkle-links
)

// Node is an IPLD node, which is {,de}serialized to CBOR or JSON)
//
//    "myfield": { "/": "Qmabcbcbdba" }
//
// The "/" key denotes a merkle-link, which IPFS handles specially.
// The object containing "/" must consist only of the "/" field, with no
// sibling elements.
type Node map[string]interface{}

// Get retrieves a property of the node. it uses unix path notation,
// splitting on "/".
func (n Node) Get(path_ string) interface{} {
	return GetPath(n, path_)
}

// Links returns all the merkle-links in the document. When the document
// is parsed, all the links are identified and references are cached, so
// getting the links only walks the document _once_. Note though that the
// entire document must be walked.
func (d Node) Links() map[string]Link {
	return Links(d)
}

// Link is a merkle-link to a target Node. The Link object is
// represented by a JSON style map:
//
//   { "/": <multihash-or-multiaddr> }
//
// Links must have only a single element (the link itself).  To associate
// additional properties with the link, you can nest the link inside a
// larger structure.
//
// looking at a whole filesystem node, we might see something like:
//
//   {
//     "foo": {
//       "unixType": "dir",
//       "unixMode": "0777",
//       "link": { "/": <multihash> }
//     },
//     "bar": {
//       "unixType": "file",
//       "unixMode": "0755",
//       "link": { "/": <multihash> }
//     }
//   }

type Link Node

// LinkStr returns the string value of l["/"],
// which is the value we use to store hashes.
func (l Link) LinkStr() string {
	s, _ := l[LinkKey].(string)
	return s
}

// Hash returns the multihash value of the link.
// TODO(yusef) add binary multiaddr parsing
func (l Link) Hash() (mh.Multihash, error) {
	s := l.LinkStr()
	if s == "" {
		return nil, errors.New("no hash in link")
	}
	return mh.FromB58String(s)
}

// Equal returns whether two Link objects are equal.
// It uses reflect.DeepEqual, so beware comparing
// large structures.
func (l Link) Equal(l2 Link) bool {
	return reflect.DeepEqual(l, l2)
}

// Links walks given node and returns all links found,
// in a flattened map. the map keys use path notation,
// made up of the intervening keys. For example:
//
// 		{
//			"foo": {
//				"quux": { "/": "Qmaaaa..." },
// 			},
//			"bar": {
//				"baz": { "/": "Qmbbbb..." },
//			},
//		}
//
// would produce links:
//
// 		{
//			"foo/quux": { "/": "Qmaaaa..." },
//			"bar/baz": { "/": "Qmbbbb..." },
//		}
//
func Links(n Node) map[string]Link {
	m := map[string]Link{}
	Walk(n, func(root, curr Node, path string, err error) error {
		if err != nil {
			return err // if anything went wrong, bail.
		}

		if l, ok := LinkCast(curr); ok {
			m[path] = l
		}
		return nil
	})
	return m
}

// checks whether a value is a link. for now we assume that all links
// follow:
//
//   { "/": "<multihash>" }
// TODO(yusef): allow binary multiaddrs with type []byte
func IsLink(v interface{}) bool {
	vn, ok := v.(Node)
	if !ok {
		return false
	}

	_, ok = vn[LinkKey].(string)
	return ok && len(vn) == 1
}

// returns the link value of an object. for now we assume that all links
// follow:
//
//   { "mlink": "<multihash>" }
func LinkCast(v interface{}) (l Link, ok bool) {
	if !IsLink(v) {
		return
	}

	l = make(Link)
	for k, v := range v.(Node) {
		l[k] = v
	}
	return l, true
}

func (n Node) Read(fun stream.ReadFun) error {
	err := read(n, fun, []interface{}{})
	if err == stream.NodeReadAbort || err == stream.NodeReadSkip {
		err = nil
	}
	return err
}

func read(curr interface{}, fun stream.ReadFun, path []interface{}) error {
	if nc, ok := curr.(Node); ok { // it's a node!
		err := fun(path, stream.TokenNode, nil)
		if err != nil {
			return err
		}

		// Iterate in fixed order (by default, go randomize iteration order)
		// Simulate reading from a file where the order is fixed
		keys := make([]string, 0, len(nc))
		for k := range nc {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			err := fun(path, stream.TokenKey, k)
			if err == stream.NodeReadSkip {
				continue
			} else if err != nil {
				return err
			}

			subpath := append(path, k)
			err = read(nc[k], fun, subpath)
			if err != nil && err != stream.NodeReadSkip {
				return err
			}
		}

		err = fun(path, stream.TokenEndNode, nil)
		if err != nil {
			return err
		}

	} else if sc, ok := curr.([]interface{}); ok { // it's a slice!
		err := fun(path, stream.TokenArray, nil)
		if err != nil {
			return err
		}

		for i, v := range sc {
			err := fun(path, stream.TokenIndex, i)
			if err == stream.NodeReadSkip {
				continue
			} else if err != nil {
				return err
			}

			subpath := append(path, i)
			err = read(v, fun, subpath)
			if err != nil && err != stream.NodeReadSkip {
				return err
			}
		}

		err = fun(path, stream.TokenEndArray, nil)
		if err != nil {
			return err
		}

	} else {
		err := fun(path, stream.TokenValue, curr)
		if err != nil {
			return err
		}
	}
	return nil
}
