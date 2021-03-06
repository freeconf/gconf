package yang

import (
	"testing"

	"github.com/freeconf/gconf/meta"
)

func TestImport(t *testing.T) {
	subYang := `
module sub {
	namespace "";
	revision 0;

	grouping x {
	  leaf sub {
	    type int32;
	  }
	}
	leaf no {
		type string;
	}
}`

	mainYang := `
module main {
	namespace "";
	import sub {
		prefix "s";
	}
	revision 0;

	grouping x {
	  leaf main {
	    type int32;
	  }
	}
	container x {
	  leaf y {
	    type int32;
	  }
	}
	uses s:x;
}`
	source := &meta.StringSource{Streamer: func(m string) (string, error) {
		switch m {
		case "main":
			return mainYang, nil
		case "sub":
			return subYang, nil
		}
		panic(m)
	}}
	m, err := LoadModule(source, "main")
	if err != nil {
		t.Error(err)
		return
	}
	if m := meta.Find(m, "sub"); m == nil {
		t.Error("Could not find s:sub container")
	}
	if m := meta.Find(m, "no"); m != nil {
		t.Error("non should not have been imported")
	}
}
