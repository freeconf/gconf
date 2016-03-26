package node

import (
	"bytes"
	"fmt"
	"meta"
	"meta/yang"
	"testing"
	"os"
)

func printMeta(m meta.Meta, level string) {
	fmt.Printf("%s%s\n", level, m.GetIdent())
	if nest, isNest := m.(meta.MetaList); isNest {
		if len(level) >= 16 {
			panic("Max level reached")
		}
		i2 := meta.NewMetaListIterator(nest, false)
		for i2.HasNextMeta() {
			printMeta(i2.NextMeta(), level+"  ")
		}
	}
}

func TestYangBrowse(t *testing.T) {
	moduleStr := `
module json-test {
	prefix "t";
	namespace "t";
	revision 0;
	list hobbies {
		container birding {
			leaf favorite-species {
				type string;
			}
		}
		container hockey {
			leaf favorite-team {
				type string;
			}
		}
	}
}`
	m, err := yang.LoadModuleCustomImport(moduleStr, nil)
	if err != nil {
		t.Fatal("bad module", err)
	}
	var actual bytes.Buffer
	c := NewContext()
	if err = c.Selector(SelectModule(m, false)).InsertInto(NewJsonWriter(&actual).Node()).LastErr; err != nil {
		t.Error(err)
	} else {
		t.Log("Round Trip:", string(actual.Bytes()))
	}
}

// TODO: support typedefs - simpleyang datatypes that use typedefs return format=0
func TestYangWrite(t *testing.T) {
	simple, err := yang.LoadModuleCustomImport(yang.TestDataSimpleYang, nil)
	if err != nil {
		t.Fatal(err)
	}
	copy := &meta.Module{}
	from := SelectModule(simple, false)
	to := SelectModule(copy, false)
	c := NewContext()
	err = c.Selector(from).UpsertInto(to.Node()).LastErr
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout.WriteString("\n*********** O R I G I N A L **********\n")
	orig, _ := os.Create("original.json")
	defer orig.Close()
	c.Selector(from).InsertInto(NewJsonWriter(orig).Node())

	os.Stdout.WriteString("\n*********** C O P Y **********\n")
	new, _ := os.Create("new.json")
	defer new.Close()
	c.Selector(to).InsertInto(NewJsonWriter(new).Node())

	// dump original and clone to see if anything is missing
	diff := Diff(from.Node(), to.Node())
	var out bytes.Buffer
	c.Selector(from.Fork(diff)).InsertInto(NewJsonWriter(&out).Node())
	t.Log(out.String())
}