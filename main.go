package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
)

func getRoot(writer http.ResponseWriter, request *http.Request) {
	fmt.Printf("got / request\n")
	http.ServeFile(writer, request, "./src/templates/index.html")
}
func getHello(writer http.ResponseWriter, request *http.Request) {
	fmt.Printf("got /hello request\n")
	http.ServeFile(writer, request, "./src/templates/hello.html")
}

func main() {
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./src/styles"))
	mux.Handle("/styles/", http.StripPrefix("/styles", fs))

	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/hello", getHello)

	err := http.ListenAndServe("127.0.0.1:3333", mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
