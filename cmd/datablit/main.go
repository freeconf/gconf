package main

import (
	"flag"
	"node"
	"os"
	"fmt"
	"meta/yang"
)

var moduleName = flag.String("module", "", "Module name (w/o *.yang extension)")

func main() {
	flag.Parse()

	// TODO: Change this to a file-persistent store.
	if len(*moduleName) == 0 {
		fmt.Fprint(os.Stderr, "Required 'module' parameter missing\n")
		os.Exit(-1)
	}

	m, err := yang.LoadModule(yang.YangPath(), *moduleName)
	if err != nil {
		panic(err)
	}
	config := node.NewJsonWriter(os.Stdout).Node()
	c := node.NewContext()
	if err != nil {
		panic(err)
	}
	if err = c.Selector(node.NewSchemaData(m, false).Select()).InsertInto(config).LastErr; err != nil {
		panic(err)
	}
}