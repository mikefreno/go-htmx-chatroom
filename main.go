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
	http.ServeFile(writer, request, getPath("/src/templates/index.html"))
}
func getChat(writer http.ResponseWriter, request *http.Request) {
	//fmt.Printf("got /chat request\n")
	http.ServeFile(writer, request, getPath("/src/templates/chat.html"))
}
func getAbout(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, getPath("/src/templates/about.html"))
}

// utilities
func cleanInput(string string) string {
	regex := regexp.MustCompile("[^a-zA-Z]+")
	regex2 := regexp.MustCompile("(?i)[^a-z]+|script")
	firstPass := regex.ReplaceAllString(string, "")
	secondPass := regex2.ReplaceAllString(firstPass, "")
	return html.EscapeString(secondPass)
}

func getPath(relativePath string) string {
	pathPrefix := os.Getenv("PATH_PREFIX")
	return fmt.Sprintf("%s/%s", pathPrefix, relativePath)
}

// api
func apiValidate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	//fmt.Printf("got api/validate request\n")
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
	cleanedName := cleanInput(params.Get("name"))
	//fmt.Println("Recieved Name: ", cleanedName)

	if cleanedName != params.Get("name") {
		htmlres := fmt.Sprintf(`<p class="fade-in mt-2 italic text-red-400">You're input contains invalid characters, reading: %s</p>`, cleanedName)
		writer.Header().Set("Content-Type", "text/html")
		fmt.Fprint(writer, htmlres)
	} else if len(cleanedName) < 3 {
		htmlres := `<p class="fade-in mt-2 italic text-red-400">Name must be at least 3 characters long</p>`
		writer.Header().Set("Content-Type", "text/html")
		fmt.Fprint(writer, htmlres)
	} else {
		htmlres := `
		<div class="flex justify-end">
			<button class="set-btn fade-in" type="submit">Set</button>
		</div>`
		writer.Header().Set("Content-Type", "text/html")
		fmt.Fprint(writer, htmlres)
	}
}

func apiNameSet(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	//fmt.Printf("got api/name-set request\n")
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	params, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	cleanedName := cleanInput(params.Get("name"))
	htmlres := fmt.Sprintf(`
		<div class="fade-in flex flex-col justify-center">
			<p>Hey, %s.</p>
			<a class="set-btn" href="/">Change Name?</a>
			<form hx-target="#body" hx-post="/api/getChannels">
				<button class="set-btn" type="submit">Join channel</button>
				<input value="%s" class="hidden" name="name">
			</form>
		</div>`, cleanedName, cleanedName)
	writer.Header().Set("Content-Type", "text/html")
	fmt.Fprint(writer, htmlres)
}
func apiGetChannels(writer http.ResponseWriter, request *http.Request) {
	htmlres := `
	<body class="page-fade-in w-full bg-zinc-50 dark:bg-zinc-800">
		<div class="fixed mx-auto border border-zinc-800 dark:border-zinc-100 bg-zinc-50 dark:bg-zinc-700 shadown-2xl rounded-sm px-8 py-6">
		    <button class="set-btn">Create New</button>
		    <h3 class="text-3xl">Populated Channels</h3>
		    <h3 class="text-3xl">Unpopulated Channels</h3>
		</div>
	</body>`
	writer.Header().Set("Content-Type", "text/html")
	fmt.Fprint(writer, htmlres)
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
	js_fs := http.FileServer(http.Dir(getPath("/src/scripts")))
	mux.Handle("/scripts/", http.StripPrefix("/scripts", js_fs))
	css_fs := http.FileServer(http.Dir(getPath("/src/styles")))
	mux.Handle("/styles/", http.StripPrefix("/styles", css_fs))

	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/chat", getChat)
	mux.HandleFunc("/about", getAbout)
	mux.HandleFunc("/chat/ws", handleConnections)
	mux.HandleFunc("/api/validate", apiValidate)
	mux.HandleFunc("/api/name-set", apiNameSet)
	mux.HandleFunc("/api/getChannels", apiGetChannels)

	go handleMessages()

	err := http.ListenAndServe(":8080", mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
