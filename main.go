package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Client struct {
	id       string
	exists   bool
	receiver chan []byte
}

type ChatRoom struct {
	clientMap map[string]*Client
	notifier  chan []byte
	chats     []string
	locker    sync.Mutex
	leaveChan chan string
}

var router = mux.NewRouter()
var chatBox *ChatRoom

func main() {
	log.Println("Starting streamChat...")
	chatBox = NewChatRoom()
	router.HandleFunc("/join", Join)
	router.HandleFunc("/leave", Leave)
	router.HandleFunc("/send", Send)
	router.HandleFunc("/message", Messages)

	startServer()

}

func NewChatRoom() *ChatRoom {
	cr := &ChatRoom{
		clientMap: make(map[string]*Client),
		notifier:  make(chan []byte, 100),
		leaveChan: make(chan string, 100),
	}
	go cr.listen()
	return cr
}

func startServer() {
	port := "localhost:7020"
	server := http.Server{
		Addr:    port,
		Handler: router,
	}
	log.Println("Serving on :", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to serve:", err)
	}
}

func Join(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	fmt.Println("Joining Client Id:", id)
	if _, ok := chatBox.clientMap[id]; ok {
		http.Error(w, "client id exists already", http.StatusOK)
		return
	}
	chatBox.clientMap[id] = &Client{id: id, exists: true, receiver: make(chan []byte, 10)}
	fmt.Fprintf(w, "Joined with Id:%s", id)
}

func Send(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if client, ok := chatBox.clientMap[id]; !ok || !client.exists {
		http.Error(w, "Access Denied. Join Chat to send messages", http.StatusOK)
		return
	}
	message := r.URL.Query().Get("message")
	chat := fmt.Sprintf("Client %s: %s", id, message)
	notify(chat)
	fmt.Fprintf(w, "Message sent")

}
func notify(data string) {
	chatBox.locker.Lock()
	defer chatBox.locker.Unlock()
	chatBox.notifier <- []byte(data)
}

func Leave(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	fmt.Println("Removing Client Id:", id)
	if _, ok := chatBox.clientMap[id]; !ok {
		http.Error(w, "ID Not found", http.StatusOK)
		return
	}
	chatBox.leaveChan <- id

	fmt.Fprintf(w, "Removed ID: %s", id)
}

func Messages(w http.ResponseWriter, r *http.Request) {

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if _, ok := chatBox.clientMap[id]; !ok {
		http.Error(w, "Access Denied. Join Chat to recieve messages", http.StatusOK)
		return
	}

	idleTimer := time.NewTimer(120 * time.Second)
	defer idleTimer.Stop()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	client := chatBox.clientMap[id]
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Stream unsupported", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s\n", strings.Join(chatBox.chats, "\n"))
	flusher.Flush()
	for client != nil {
		select {

		case event, ok := <-client.receiver:
			if !ok {
				fmt.Println("client channel closed..")
				return
			}

			fmt.Fprintf(w, "%s\n", event)
			flusher.Flush()

			if !idleTimer.Stop() {
				<-idleTimer.C
			}
			idleTimer.Reset(120 * time.Second)

		case <-idleTimer.C:
			fmt.Println("No activity...")
			fmt.Fprintf(w, "\nNo Recent messages.closing connection..\n\n")
			return

		case <-r.Context().Done():
			w.Write([]byte("Connection Timeout"))
			return
		}
	}

}

func (cr *ChatRoom) listen() {
	for {
		select {

		case id := <-chatBox.leaveChan:
			chatBox.locker.Lock()
			client, ok := cr.clientMap[id]
			if ok {
				close(client.receiver)
				delete(cr.clientMap, id)
				fmt.Printf("Client Left:%s\n", id)
			}
			chatBox.locker.Unlock()

		case event := <-chatBox.notifier:
			fmt.Println("Message:", string(event))
			chatBox.chats = append(chatBox.chats, string(event))
			for client := range chatBox.clientMap {
				chatBox.clientMap[client].receiver <- event
			}
		}
	}
}
