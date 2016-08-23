package memory

import (
	"testing"

	mh "github.com/jbenet/go-multihash"
)

type TC struct {
	src   Node
	links map[string]string
}

var testCases []TC

func mmh(b58 string) mh.Multihash {
	h, err := mh.FromB58String(b58)
	if err != nil {
		panic("failed to decode multihash")
	}
	return h
}

func init() {
	testCases = append(testCases, TC{
		src: Node{
			"foo": "bar",
			"bar": []int{1, 2, 3},
			"baz": Node{
				"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
			},
			// FIXME: This should be disallowed, since the top-level uses a "/" key but its value is a Node,
			// not a link.  As is, the double / will be collapsed by the path traversal, and the link will be
			// accessible at "/test"
			"test": Node{
				"/": Node{
					"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
				},
			},
		},
		links: map[string]string{
			"baz":        "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
		},
	}, TC{
		src: Node{
			"baz": Node{
				"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
			},
			"bazz": Node{
				"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
			},
			"bar": Node{
				"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPb",
			},
			"bar2": Node{
				"@bar": Node{
					"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPa",
				},
				"\\@foo": Node{
					"/": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPa",
				},
			},
		},
		links: map[string]string{
			"baz":       "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
			"bazz":      "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPo",
			"bar":       "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPb",
			"bar2/@bar": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPa",
			"bar2/\\@foo": "QmZku7P7KeeHAnwMr6c4HveYfMzmtVinNXzibkiNbfDbPa",
		},
	})
}

func TestParsing(t *testing.T) {
	for tci, tc := range testCases {
		t.Logf("===== Test case #%d =====", tci)
		doc := tc.src

		// check links
		links := doc.Links()
		t.Logf("links: %#v", links)
		if len(links) != len(tc.links) {
			t.Errorf("links do not match, not the same number of links, expected %d, got %d", len(tc.links), len(links))
		}
		for k, l1 := range tc.links {
			l2 := links[k]
			if l1 != l2["/"] {
				t.Errorf("links do not match. %d/%#v %#v != %#v[/]", tci, k, l1, l2)
			}
		}
	}
}
