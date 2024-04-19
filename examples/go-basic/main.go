package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	// this could be replaced by some middleware that reflects on routes,
	// but you can't enumerate patterns with the built-in ServeMux. most
	// other routers like github.com/gorilla/mux let you do this, but here
	// is a manual example anyway.
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/vnd.httpfs.v1+json")
		b, _ := json.Marshal(map[string]any{
			"dir": []any{
				"hello",
			},
		})
		w.Write(b)
	}))

	http.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello world!\n")
	}))

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	fmt.Println("listening on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
