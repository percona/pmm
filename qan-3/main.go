package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
} // use default options

func listener(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)

		response := router(string(message))

		err = c.WriteMessage(mt, []byte(response))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func router(message string) string {
	data, err := os.ReadFile(fmt.Sprintf("./templates/%s.html", message))
	if err != nil {
		return err.Error()
	}

	return string(data)
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", listener)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
