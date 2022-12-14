package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
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
	log.Fatal(http.ListenAndServe(*addr, nil))
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

		var body request
		json.Unmarshal(message, &body)

		bytes, err := json.Marshal(router(body.Kind))
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

func router(name string) response {
	target, html, err := readHTMLFile(fmt.Sprintf("./html/%s.html", name))
	if err != nil {
		return response{
			Error: err.Error(),
		}
	}

	script, err := os.ReadFile(fmt.Sprintf("./scripts/%s.js", name))
	if err != nil {
		// script is optional.
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

	target := ""
	html := []string{}
	lineNumber := 0
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		lineNumber++

		if lineNumber == 1 {
			line := fileScanner.Text()
			fmt.Println(line)
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
