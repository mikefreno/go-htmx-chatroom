package main

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

var upgrader = websocket.Upgrader{}

type Message struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// pages
func getRoot(writer http.ResponseWriter, request *http.Request) {
	//fmt.Printf("got / request\n")
	http.ServeFile(writer, request, "./src/templates/index.html")
}
func getChat(writer http.ResponseWriter, request *http.Request) {
	//fmt.Printf("got /chat request\n")
	http.ServeFile(writer, request, "./src/templates/chat.html")
}

// api
func apiNameSet(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("got api/name-set request\n")
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	fmt.Println("Body:", string(body))
}

func apiValidate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("got api/validate request\n")
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	params, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(writer, "Error parsing body", http.StatusInternalServerError)
		return
	}
	name := params.Get("name")
	regex := regexp.MustCompile("[^a-zA-Z]+")
	regex2 := regexp.MustCompile("(?i)[^a-z]+|script")
	name = regex.ReplaceAllString(name, "")
	name = regex2.ReplaceAllString(name, "")
	cleanedName := html.EscapeString(name)

	fmt.Println("Recieved Name: ", cleanedName)
}

// websocket
func handleConnections(writer http.ResponseWriter, request *http.Request) {
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ws.Close()

	clients[ws] = true

	for {
		var msg Message

		err := ws.ReadJSON(&msg)
		if err != nil {
			fmt.Println(err)
			delete(clients, ws)
			break
		}

		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {
	mux := http.NewServeMux()

	js_fs := http.FileServer(http.Dir("./src/scripts"))
	mux.Handle("/scripts/", http.StripPrefix("/scripts", js_fs))
	css_fs := http.FileServer(http.Dir("./src/styles"))
	mux.Handle("/styles/", http.StripPrefix("/styles", css_fs))

	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/chat", getChat)
	mux.HandleFunc("/chat/ws", handleConnections)
	mux.HandleFunc("/api/validate", apiValidate)
	mux.HandleFunc("/api/name-set", apiNameSet)

	go handleMessages()

	err := http.ListenAndServe("127.0.0.1:3333", mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
