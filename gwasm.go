//go:build js
// +build js

package gwasm

import (
	"io"
	"strings"
	"syscall/js"
	"time"
)

// AddScript is a helper function for adding a script to HTML document head
// and verifying it was added correctly given the object name.
// If no object name is passed the script will not wait
func AddScript(url, objName string, timeout time.Duration) {
	switch {
	case objName == "":
		panic("need an object name/namespace to verify it is available!")
	case timeout <= 0:
		panic("need greater-than zero timeout")
	case strings.Contains(objName, " "):
		panic("objName must be a javascript identifier (no spaces)")
	case !js.Global().Get(objName).IsUndefined():
		panic("objName is already defined in global space")
	}
	script := js.Global().Get("document").Call("createElement", "script")
	script.Set("src", url)
	js.Global().Get("document").Get("head").Call("appendChild", script)
	start := time.Now()
	for {
		time.Sleep(60 * time.Millisecond)
		if jsObject := js.Global().Get(objName); !jsObject.IsUndefined() {
			break
		} else if time.Since(start) > timeout {
			panic("timeout while obtaining " + objName + " from URL: " + url)
		}
	}
}

// DownloadStream downloads an io.Reader's contents
// as a file with the contentType (i.e. "text/csv", "application/json")
func DownloadStream(filename, contentType string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	opts := js.Global().Get("Object").New()
	if contentType != "" {
		opts.Set("type", contentType)
	} else {
		opts.Set("type", "application/octet-stream")
	}

	barray := js.Global().Get("Array").New(string(b))
	blob := js.Global().Get("Blob")
	blob = blob.New(barray, opts)

	tempAnchor := js.Global().Get("document").Call("createElement", "a")
	tempAnchor.Set("href", js.Global().Get("URL").Call("createObjectURL", blob))
	tempAnchor.Set("download", filename)
	js.Global().Get("document").Get("body").Call("appendChild", tempAnchor)
	tempAnchor.Call("click")
	js.Global().Get("document").Get("body").Call("removeChild", tempAnchor)
	return nil
}
