package nodes_test

import (
	"flag"
	"testing"

	"github.com/freeconf/gconf/c2"
	"github.com/freeconf/gconf/meta"
	"github.com/freeconf/gconf/meta/yang"
	"github.com/freeconf/gconf/nodes"
)

var updateFlag = flag.Bool("update", false, "Update the golden files.")

func TestSchemaRead(t *testing.T) {
	tests := []string{
		"json-test",
		"choice",
	}
	for _, test := range tests {
		m := yang.RequireModule(&meta.FileStreamSource{Root: "./testdata"}, test)
		ypath := &meta.FileStreamSource{Root: "../yang"}
		ymod := yang.RequireModule(ypath, "yang")
		sel := nodes.Schema(ymod, m).Root()
		actual, err := nodes.WritePrettyJSON(sel)
		if err != nil {
			t.Error(err)
		}
		c2.Gold(t, *updateFlag, []byte(actual), "gold/"+test+".schema.json")
	}
}
