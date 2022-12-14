package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", "localhost:8080", "http service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type response struct {
	HTML   string
	Script string
	Error  string
}

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

		bytes, err := json.Marshal(router(message))
		if err != nil {
			log.Println("write:", err)
			break
		}

		err = c.WriteMessage(mt, bytes)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func router(message []byte) response {
	html, err := os.ReadFile(fmt.Sprintf("./html/%s.html", message))
	if err != nil {
		return response{
			Error: err.Error(),
		}
	}

	script, err := os.ReadFile(fmt.Sprintf("./scripts/%s.js", message))
	if err != nil {
		// script is optional.
		if !errors.Is(err, os.ErrNotExist) {
			return response{
				Error: err.Error(),
			}
		}
	}

	return response{
		HTML:   string(html),
		Script: string(script),
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", listener)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
