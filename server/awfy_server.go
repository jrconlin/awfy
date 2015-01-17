package main

import (
	"bytes"
	"flag"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
)

const DATA_SIZE = 1024
const CHANNELS = 256

var (
	addr      = flag.String("addr", ":8080", "http service address")
	tmplPath  = flag.String("templates", "tmpl", "path to templates")
	homeTempl *template.Template
)

type hub struct {
	connections map[*connection]bool
	broadcast   chan []byte
	register    chan *connection
	unregister  chan *connection
}

var h = hub{
	broadcast:   make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
}

func trimSpace(in []byte) (out []byte) {
	const sp byte = byte(' ')
	const quote byte = byte('"')
	i := 0
	prev := sp
	inq := false
	outb := make([]byte, len(in))
	for _, c := range []byte(in) {
		if c == quote {
			inq = !inq
		}
		if !inq && c == sp && c == prev {
			continue
		}
		outb[i] = c
		i++
		prev = c
	}
	if len(outb) > 0 && outb[i-1] == sp {
		outb[i-1] = 0
	}
	return outb
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			log.Printf("Registering...\n")
			h.connections[c] = true
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				log.Printf("UnRegistering...\n")
				delete(h.connections, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			log.Printf("Broadcasting...\n")
			m = bytes.Map(func(r rune) rune {
				if r < ' ' {
					return -1
				}
				return r
			}, m)
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					delete(h.connections, c)
					close(c.send)
				}
			}
		}
	}
}

type connection struct {
	ws   *websocket.Conn
	send chan []byte
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			break
		}
		h.broadcast <- message
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) Count() int {
	return len(c.send)
}

var upgrader = &websocket.Upgrader{ReadBufferSize: DATA_SIZE,
	WriteBufferSize: DATA_SIZE}

func wsHandler(resp http.ResponseWriter, req *http.Request) {
	ws, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {
		return
	}
	log.Printf("Connect")
	c := &connection{
		send: make(chan []byte, CHANNELS),
		ws:   ws,
	}
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader()
}

func main() {
	flag.Parse()

	go h.run()
	http.HandleFunc("/metrics", func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("Sorry, no metrics yet..."))
	})
	http.HandleFunc("/", wsHandler)
	log.Printf("Starting up server at %s\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("Could not start server:", err)
	}

}
