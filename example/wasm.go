//go:build js
// +build js

package main

import "syscall/js"

func main() {
	p := createElement("p")
	p.Set("innerText", "Hello World!!")
	js.Global().Get("document").Get("body").Call("appendChild", p)
}

func createElement(e string) js.Value {
	return js.Global().Get("document").Call("createElement", e)
}
