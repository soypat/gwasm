//go:build !js
// +build !js

package gwasm

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Taken from https://github.com/hajimehoshi/wasmserve with slight modifications

const indexHTML = `<!DOCTYPE html>
<!-- Polyfill for the old Edge browser -->
<script src="https://cdn.jsdelivr.net/npm/text-encoding@0.7.0/lib/encoding.min.js"></script>
<script src="wasm_exec.js"></script>
<script>
(async () => {
  const resp = await fetch('main.wasm');
  if (!resp.ok) {
    const pre = document.createElement('pre');
    pre.innerText = await resp.text();
    document.body.appendChild(pre);
  } else {
    const src = await resp.arrayBuffer();
    const go = new Go();
    const result = await WebAssembly.instantiate(src, go.importObject);
    go.argv = [];
    go.run(result.instance);
  }
  const reload = await fetch('_wait');
  // The server sends a response for '_wait' when a request is sent to '_notify'.
  if (reload.ok) {
    location.reload();
  }
})();
</script>
`

// WASMHandler serves a WASM application. Implements http.Handler interface.
type WASMHandler struct {
	IndexHTML       string
	WASMReload      bool
	startTime       time.Time
	wasmDir         string
	wasmcontent     []byte
	wasmModTime     time.Time
	wasmexeccontent []byte
	tmpOutputDir    string
	output          io.Writer
	waitChannel     chan struct{}
	subHandler      http.HandlerFunc
}

// NewWASMHandler returns a handler which does basically the same thing as https://github.com/hajimehoshi/wasmserve.
//
// Example of usage which does most of what `wasmserve` does:
//  wsm, err := gwasm.NewWASMHandler("app", nil)
//  if err != nil {
//  	log.Fatal(err)
//  }
//  wsm.WASMReload = true
//  wsm.SetOutput(os.Stdout)
//  http.Handle("/", wsm)
//  log.Fatal(http.ListenAndServe(":8080", nil))
func NewWASMHandler(wasmDir string, subHandler http.HandlerFunc) (*WASMHandler, error) {
	var err error
	wsm := &WASMHandler{
		wasmDir:     wasmDir,
		startTime:   time.Now(),
		waitChannel: make(chan struct{}),
		subHandler:  subHandler,
	}
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return nil, err
	}
	f := filepath.Join(strings.TrimSpace(string(out)), "misc", "wasm", "wasm_exec.js")
	wsm.wasmexeccontent, err = readFile(f)
	if err != nil {
		return nil, err
	}
	err = wsm.setTmpOutputDir()
	if err != nil {
		return nil, err
	}
	wsm.IndexHTML = indexHTML
	err = wsm.buildWASM()
	if err != nil {
		return nil, err
	}
	return wsm, nil
}

// ServeHTTP implements http.Handler interface. For use with http.Handle
func (wsm *WASMHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path[1:]
	fpath := path.Base(upath)
	if !strings.HasSuffix(r.URL.Path, "/") {
		fi, err := os.Stat(fpath)
		if err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fi != nil && fi.IsDir() {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusSeeOther)
			return
		}
	}
	baseFpath := filepath.Base(fpath)
	switch baseFpath {
	case ".", "index.html":
		http.ServeContent(w, r, "index.html", time.Now(), strings.NewReader(wsm.IndexHTML))
		return
	case "wasm_exec.js":
		http.ServeContent(w, r, "wasm_exec.js", wsm.startTime, bytes.NewReader(wsm.wasmexeccontent))
		return
	case "main.wasm":
		if wsm.WASMReload {
			err := wsm.buildWASM()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if len(wsm.wasmcontent) == 0 {
			http.Error(w, "no wasm content", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, "main.wasm", wsm.wasmModTime, bytes.NewReader(wsm.wasmcontent))
		return
	case "_wait":
		wsm.waitForUpdate(w, r)
		return
	case "_notify":
		wsm.notifyWaiters(w, r)
		return
	}

	if wsm.subHandler != nil {
		wsm.subHandler(w, r)
	} else {
		msg := "\"" + fpath + "\" path not found\n"
		wsm.log([]byte(msg))
		http.Error(w, msg, 404)
	}
}

func (wsm *WASMHandler) setTmpOutputDir() (err error) {
	wsm.tmpOutputDir, err = ioutil.TempDir("", "")
	return err
}

func (wsm *WASMHandler) buildWASM() error {
	buildName := filepath.Join(wsm.tmpOutputDir, "main.wasm")
	args := []string{"build", "-o", buildName}
	wsm.log([]byte("go " + strings.Join(args, " ")))
	cmdBuild := exec.Command("go", args...)
	cmdBuild.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmdBuild.Dir = wsm.wasmDir
	out, err := cmdBuild.CombinedOutput()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		wsm.log(out)
	}
	wasmContent, err := readFile(buildName)
	if err != nil {
		return err
	}
	wsm.wasmcontent = wasmContent
	wsm.wasmModTime = time.Now()
	return nil
}

// SetOutput sets logging write output to visualize compile-time errors and bad requests.
func (wsm *WASMHandler) SetOutput(w io.Writer) { wsm.output = w }

func (wsm *WASMHandler) log(b []byte) {
	if wsm.output != nil {
		wsm.output.Write(b)
		if b[len(b)-1] != '\n' {
			wsm.output.Write([]byte{'\n'})
		}
	}
}

func (wsm *WASMHandler) waitForUpdate(w http.ResponseWriter, r *http.Request) {
	wsm.waitChannel <- struct{}{}
	http.ServeContent(w, r, "", time.Now(), bytes.NewReader(nil))
}

func (wsm *WASMHandler) notifyWaiters(w http.ResponseWriter, r *http.Request) {
	for {
		select {
		case <-wsm.waitChannel:
		default:
			http.ServeContent(w, r, "", time.Now(), bytes.NewReader(nil))
			return
		}
	}
}

func readFile(filename string) ([]byte, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	return ioutil.ReadAll(fp)
}
