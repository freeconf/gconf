package main

import (
	"flag"

	"github.com/freeconf/gconf/gateway"
	"github.com/freeconf/gconf/restconf"

	"github.com/freeconf/gconf/meta"

	"github.com/freeconf/gconf/device"
	"github.com/freeconf/gconf/meta/yang"

	"github.com/freeconf/gconf/c2"
)

// Management Gateway.  Serve management functions to available services.
//
// Then open web browser to
//   http://localhost:8080/
//

var startup = flag.String("startup", "startup.json", "startup configuration file.")
var verbose = flag.Bool("verbose", false, "verbose")
var web = flag.String("web", "", "web directory")
var varDir = flag.String("var", "var", "directory to store files")

func main() {
	flag.Parse()
	c2.DebugLog(*verbose)

	// where all yang files are stored
	ypath := yang.YangPath()

	// Even though this is a server component, we still organize things thru a device
	// because this proxy will appear like a "Device" to application management systems
	// "northbound"" representing all the devices that are "southbound".
	var d *device.Local
	if *web == "" {
		d = device.New(ypath)
	} else {
		d = device.NewWithUi(ypath, &meta.FileStreamSource{Root: *web})
	}

	// We "wrap" each device with a device that splits CRUD operations
	// to local store AND the original device.  This gives us transparent
	// persistance of device data w/o altering the device API.
	reg := gateway.NewLocalRegistrar()
	m := gateway.NewFileStore(reg, "var")
	gateway.NewService(d, m, reg)

	// Add RESTCONF service, if you had other protocols to add/replace
	// you could do that here
	mgmt := restconf.NewServer(d)

	// Let RESTCONF know it's proxy for registered devices
	mgmt.ServeDevices(m)

	// bootstrap config for all local modules
	chkErr(d.ApplyStartupConfigFile(*startup))

	// Wait for cntrl-c...
	select {}
}

func chkErr(err error) {
	if err != nil {
		panic(err)
	}
}
