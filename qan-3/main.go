package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	htmlCommentStart = "<!--"
	htmlCommentEnd   = "-->"
)

var (
	addr     = flag.String("addr", "localhost:8080", "http service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type request struct {
	Kind string
	Data string
}

type response struct {
	Target string
	HTML   string
	Script string
	Error  string
}

func main() {
	http.HandleFunc("/", listener)
	http.HandleFunc("/fe.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "fe.js") })
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func listener(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Println("recv:", string(message))

		var body request
		json.Unmarshal(message, &body)

		u, err := url.Parse(r.Header.Get("Origin"))
		if err != nil {
			log.Println("read:", err)
			break
		}

		bytes, err := json.Marshal(router(u.Host, body.Kind, body.Data))
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

func router(origin, kind, data string) response {
	log.Println("origin:", origin)

	switch kind {
	case "get":
		return get(origin, kind)
	default:
		return response{
			Error: "not supported kind",
		}
	}
}

func get(origin, kind string) response {
	// TODO check ../ etc
	// TODO format origin to path friendly
	target, html, err := readHTMLFile(fmt.Sprintf("./html/%s.html", origin))
	if err != nil {
		// html/css is optional
		if !errors.Is(err, os.ErrNotExist) {
			return response{
				Error: err.Error(),
			}
		}
	}

	script, err := os.ReadFile(fmt.Sprintf("./scripts/%s.js", origin))
	if err != nil {
		// script is optional
		if !errors.Is(err, os.ErrNotExist) {
			return response{
				Error: err.Error(),
			}
		}
	}

	return response{
		Target: target,
		HTML:   string(html),
		Script: string(script),
	}
}

func readHTMLFile(path string) (string, string, error) {
	readFile, err := os.Open(path)
	if err != nil {
		return "", "", err
	}

	var target string
	var html []string
	lineNumber := 0
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		lineNumber++

		if lineNumber == 1 {
			line := fileScanner.Text()
			if !strings.Contains(line, htmlCommentStart) || !strings.Contains(line, htmlCommentEnd) {
				return "", "", fmt.Errorf("file %s doesnt contains target on first line", path)
			}
			target = strings.Replace(strings.Replace(line, htmlCommentStart, "", 1), htmlCommentEnd, "", 1)

			continue
		}

		html = append(html, fileScanner.Text())
	}
	readFile.Close()

	return target, strings.Join(html, ""), nil
}
