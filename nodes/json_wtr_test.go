package nodes

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"github.com/freeconf/gconf/meta/yang"
	"github.com/freeconf/gconf/node"

	"github.com/freeconf/gconf/val"

	"github.com/freeconf/gconf/c2"
	"github.com/freeconf/gconf/meta"
)

func TestJsonWriterLeafs(t *testing.T) {
	c2.DebugLog(true)
	tests := []struct {
		Yang     string
		Val      val.Value
		expected string
		enumAsId bool
	}{
		{
			Yang:     `leaf-list x { type string;}`,
			Val:      val.StringList([]string{"a", "b"}),
			expected: `"x":["a","b"]`,
		},
		{
			Yang:     `leaf x { type union { type int32; type string;}}`,
			Val:      val.String("a"),
			expected: `"x":"a"`,
		},
		{
			Yang:     `leaf x { type union { type int32; type string;}}`,
			Val:      val.Int32(99),
			expected: `"x":99`,
		},
		{
			Yang:     `leaf x { type enumeration { enum zero; enum one; }}`,
			Val:      val.Enum{Id: 0, Label: "zero"},
			expected: `"x":"zero"`,
		},
		{
			Yang:     `leaf x { type enumeration { enum five {value 5;} enum six; }}`,
			Val:      val.Enum{Id: 6, Label: "six"},
			expected: `"x":6`,
			enumAsId: true,
		},
	}
	for _, test := range tests {
		m := yang.RequireModuleFromString(nil, fmt.Sprintf(`module m { namespace ""; %s }`, test.Yang))
		var actual bytes.Buffer
		buf := bufio.NewWriter(&actual)
		w := &JSONWtr{
			_out:      buf,
			EnumAsIds: test.enumAsId,
		}
		w.writeValue(m.DataDefs()[0], test.Val)
		buf.Flush()
		c2.AssertEqual(t, test.expected, actual.String())
	}
}

func TestJsonWriterListInList(t *testing.T) {
	moduleStr := `
module m {
	prefix "t";
	namespace "t";
	revision 0000-00-00 {
		description "x";
	}
	typedef td {
		type string;
	}
	list l1 {
		list l2 {
		    key "a";
			leaf a {
				type td;
			}
			leaf b {
			    type string;
			}
		}
	}
}
	`
	m := yang.RequireModuleFromString(nil, moduleStr)
	root := map[string]interface{}{
		"l1": []map[string]interface{}{
			map[string]interface{}{"l2": []map[string]interface{}{
				map[string]interface{}{
					"a": "hi",
					"b": "bye",
				},
			},
			},
		},
	}
	b := ReflectChild(root)
	sel := node.NewBrowser(m, b).Root()
	actual, err := WriteJSON(sel)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"l1":[{"l2":[{"a":"hi","b":"bye"}]}]}`
	if actual != expected {
		t.Errorf("\nExpected:%s\n  Actual:%s", expected, actual)
	}
}

func TestJsonAnyData(t *testing.T) {
	tests := []struct {
		anything interface{}
		expected string
	}{
		{
			anything: map[string]interface{}{
				"a": "A",
				"b": "B",
			},
			expected: `"x":{"a":"A","b":"B"}`,
		},
		{
			anything: []interface{}{
				map[string]interface{}{
					"a": "A",
				},
				map[string]interface{}{
					"b": "B",
				},
			},
			expected: `"x":[{"a":"A"},{"b":"B"}]`,
		},
	}
	for _, test := range tests {
		var actual bytes.Buffer
		buf := bufio.NewWriter(&actual)
		w := &JSONWtr{
			_out: buf,
		}
		m := meta.NewLeaf(nil, "x")
		w.writeValue(m, val.Any{Thing: test.anything})
		buf.Flush()
		c2.AssertEqual(t, test.expected, actual.String())
	}
}
