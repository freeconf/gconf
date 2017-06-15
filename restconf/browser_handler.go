package restconf

import (
	"mime"
	"net/http"

	"context"

	"github.com/c2stack/c2g/device"
	"github.com/c2stack/c2g/meta"
	"github.com/c2stack/c2g/node"
)

type browserHandler struct {
	browser *node.Browser
}

func (self *browserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var payload node.Node
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if r.RemoteAddr != "" {
		host, _ := ipAddrSplitHostPort(r.RemoteAddr)
		ctx = context.WithValue(ctx, device.RemoteIpAddressKey, host)
	}
	sel := self.browser.RootWithContext(ctx)
	if sel = sel.FindUrl(r.URL); sel.LastErr == nil {
		if sel.IsNil() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if handleErr(err, w) {
			return
		}
		switch r.Method {
		case "DELETE":
			err = sel.Delete()
		case "GET":
			w.Header().Set("Content-Type", mime.TypeByExtension(".json"))
			if meta.IsNotification(sel.Meta()) {
				var sub node.NotifyCloser
				wait := make(chan struct{})
				sub, err = sel.Notifications(func(msg node.Selection) {
					output := node.NewJsonWriter(w).Node()
					if err := msg.InsertInto(output).LastErr; err != nil {
						handleErr(err, w)
						return
					}
					wait <- struct{}{}
				})
				<-wait
				sub()
			} else {
				output := node.NewJsonWriter(w).Node()
				err = sel.InsertInto(output).LastErr
			}
		case "PUT":
			err = sel.UpsertFrom(node.NewJsonReader(r.Body).Node()).LastErr
		case "POST":
			if meta.IsAction(sel.Meta()) {
				a := sel.Meta().(*meta.Rpc)
				var input node.Node
				if a.Input != nil {
					input = node.NewJsonReader(r.Body).Node()
				}
				if outputSel := sel.Action(input); !outputSel.IsNil() && a.Output != nil {
					w.Header().Set("Content-Type", mime.TypeByExtension(".json"))
					err = outputSel.InsertInto(node.NewJsonWriter(w).Node()).LastErr
				} else {
					err = outputSel.LastErr
				}
			} else {
				payload = node.NewJsonReader(r.Body).Node()
				err = sel.InsertFrom(payload).LastErr
			}
		case "OPTIONS":
			// NOP
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	} else {
		err = sel.LastErr
	}

	if err != nil {
		handleErr(err, w)
	}
}