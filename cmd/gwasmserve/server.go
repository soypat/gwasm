package main

import (
	"log"
	"net/http"
	"os"

	"github.com/soypat/gwasm"
)

func main() {
	wsm, err := gwasm.NewWASMHandler("app", nil)
	if err != nil {
		log.Fatal(err)
	}
	wsm.WASMReload = true
	wsm.SetOutput(os.Stdout)
	http.Handle("/", wsm)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
