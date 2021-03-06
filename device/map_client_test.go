package device

import (
	"testing"

	"github.com/freeconf/gconf/meta"
	"github.com/freeconf/gconf/meta/yang"
	"github.com/freeconf/gconf/node"
	"github.com/freeconf/gconf/nodes"
)

func TestMapClient(t *testing.T) {
	ypath := meta.MultipleSources(
		&meta.FileStreamSource{Root: "./testdata"},
		&meta.FileStreamSource{Root: "../yang"},
	)
	d := New(ypath)
	d.Add("test", &nodes.Basic{})
	dm := NewMap()
	dm.Add("dev0", d)
	dmMod := yang.RequireModule(ypath, "map")
	dmNode := MapNode(dm)
	dmClient := &MapClient{
		proto: func(string) (Device, error) {
			return d, nil
		},
		browser: node.NewBrowser(dmMod, dmNode),
	}
	var gotUpdate bool
	dmClient.OnModuleUpdate("test", func(d Device, id string, c Change) {
		gotUpdate = true
	})
	if !gotUpdate {
		t.Error("never got test message")
	}
}
