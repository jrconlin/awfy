/* Based off of the gorilla websocket sample.
   Heavily modified by me

   Do whatever the hell you want with this code, I don't really care.
*/

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strings"
	"time"
)

const DATA_SIZE = 100
const CHANNELS = 256

var (
	addr   = flag.String("addr", ":8080", "http service address")
	dbPath = flag.String("db", "./awfy.db", "path to templates")
	debug  = flag.Bool("debug", false, "debugging output")
	db     *sql.DB
)

// Crappy logging function (only if -debug specified)
func Log(str string, arg ...interface{}) {
	if *debug {
		if arg != nil {
			Error(fmt.Sprintf(str, arg))
		} else {
			Error(str)
		}
	}
}

func Error(str string, arg ...interface{}) {
	if arg != nil {
		log.Printf(str, arg)
	} else {
		log.Printf(str)
	}
}

// Trim up spaces (including inline) in byte arrays
func trimSpace(in []byte) (out string) {
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
	return string(outb)
}

// Distribution hub
type hub struct {
	connections map[*connection]bool
	broadcast   chan []byte
	register    chan *connection
	unregister  chan *connection
	db          *sql.DB
}

var h = hub{
	broadcast:   make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
}

func (h *hub) Count() int {
	return len(h.connections)
}

func (h *hub) run(st *store) {
	h.db = db
	for {
		select {
		case c := <-h.register:
			Log("Registering...\n")
			h.connections[c] = true
			c.stcmd = st.cmd
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				Log("UnRegistering...\n")
				delete(h.connections, c)
				close(c.cmd)
			}
		case m := <-h.broadcast:
			Log("Broadcasting...\n")
			m = bytes.Map(func(r rune) rune {
				if r < ' ' {
					return -1
				}
				return r
			}, m)
			for c := range h.connections {
				c.cmd <- m
			}
		}
	}
}

// storage functions
// Using sqlite for now.

type usage struct {
	Count   int64            `json:"count"`
	Clients map[string]int64 `json:"clients"`
}

type reset_response struct {
		Type     string `json:"type"`
		Last     int64  `json:"last"`
		Previous int64  `json:"previous"`
}

type store struct {
	db  *sql.DB
	cmd chan []byte // this probably should be a struct{cmd, args, err}
}

func (r *store) getInfo(t string) (reply []byte) {
	var err error
	// read latest info
	row := r.db.QueryRow("select coalesce(time,0), coalesce(previous,0) from resets order by time desc limit 1;")
	now := time.Now().UTC().Unix() / 60
	var lastReset int64

	var resp reset_response

	err = row.Scan(&lastReset, &resp.Previous)
	switch {
	case err == sql.ErrNoRows:
		Log("No data yet...")
	case err != nil:
		Error("ERROR: getInfo: %s", err.Error())
		return
	}

	if lastReset != 0 {
		resp.Last = now - lastReset
	}
	resp.Type = t
	reply, err = json.Marshal(resp)
	if err != nil {
		Error("ERROR: getInfo: %s", err.Error())
	}
	return
}

func (r *store) reset(ua []byte) (reply []byte) {
	var err error
	stmt := `insert or abort into resets (time, previous, ua) values (strftime('%s','now')/60, max(0, (select strftime('%s','now')/60 - time from resets order by time desc limit 1)), ?);`
	Log("Resetting...")
	_, err = r.db.Exec(stmt, string(ua))
	if err != nil {
		Error("ERROR: reset: %s", err.Error())
		return reply
	}
	return r.getInfo("r")
}

func (r *store) run() {
	for {
		select {
		case c := <-r.cmd:
			if len(c) == 0 {
				continue
			}
			us := string(c)
			switch us[:1] {
			case "i":
				r.cmd <- r.getInfo(us[:1])
			case "r":
				r.cmd <- r.reset(c[1:])
			case "s":
				r.cmd <- r.getMetrics()
			}
		}
	}
}

func incr(key string, arr map[string]int64) {
	if _, ok := arr[key]; ok {
		arr[key] += 1
		return
	}
	arr[key] = 1
}

func (r *store) getMetrics() (reply []byte) {
	var err error
	r.db.Exec(`delete from resets where time < date('now', 'start of day' '-7 day');`)
	result := &usage{0, make(map[string]int64)}
	stmt := `select ua from resets where time > date('now','start of day');`
	rows, err := r.db.Query(stmt)
	defer rows.Close()
	if err != nil {
		Error("ERROR: bad stats %s", err.Error())
		return
	}
	var ua string
	for rows.Next() {
		if err = rows.Scan(&ua); err == nil {
			Error("ERROR: bad stat scan %s", err.Error())
			return
		}
		result.Count++
		switch {
		case strings.Contains(ua, "Gecko"):
			incr("gecko", result.Clients)
		case strings.Contains(ua, "Chrome"):
			incr("chrome", result.Clients)
		case strings.Contains(ua, "AppleWebKit"):
			incr("ios", result.Clients)
		default:
			incr("other", result.Clients)
		}
	}
	reply, err = json.Marshal(result)
	if err != nil {
		Error("ERROR: stat marshal %s", err.Error)
	}
	return
}

// connection worker
type connection struct {
	ws    *websocket.Conn
	cmd   chan []byte
	stcmd chan []byte
	db    *sql.DB
	ua    string
}

func (c *connection) reader() {
	defer c.ws.Close()
	for {
		_, raw, err := c.ws.ReadMessage()
		if err != nil {
			Error("ERROR in reader: %s", err.Error())
			return
		}
		message := trimSpace(raw)
		if len(message) == 0 {
			continue
		}
		switch strings.ToLower(message[:1]) {
		case "i":
			c.stcmd <- []byte("i")
			res := <-c.stcmd
			if len(res) > 0 {
				c.ws.WriteMessage(websocket.TextMessage, res)
			}
			continue
		case "r":
			c.stcmd <- []byte("r" + c.ua)
			res := <-c.stcmd
			Log("reset: %s", res)
			if len(res) > 0 {
				h.broadcast <- res
			}
		case "q":
			Log("Closing...")
			return
		default:
			Log(" WARN: bad command %s", message)
			return
		}
	}
}

func (c *connection) writer() {
	for command := range c.cmd {
		err := c.ws.WriteMessage(websocket.TextMessage, command)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) Count() int {
	return len(c.cmd)
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  DATA_SIZE,
	WriteBufferSize: DATA_SIZE,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func wsHandler(resp http.ResponseWriter, req *http.Request) {
	Log("Handling... ")
	ws, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {
		Log("upgrade failed: %s", err.Error())
		return
	}
	ua := req.Header.Get("User-Agent")
	c := &connection{
		cmd: make(chan []byte, CHANNELS),
		ws:  ws,
		ua:  ua,
	}
	Log("Connecting %s", req.RemoteAddr, ua)
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader()
}

func dbinit(db *sql.DB) (err error) {
	_, err = db.Exec(`create table if not exists resets (time integer primary key, previous integer, ua text);`)
	return err
}

func main() {
	var err error
	flag.Parse()

	born := time.Now().UTC()

	Log("Opening db at: %s", *dbPath)

	if db, err = sql.Open("sqlite3", *dbPath); err != nil {
		log.Fatal("Could not open db: %s", err.Error())
	}
	defer db.Close()

	if err = dbinit(db); err != nil {
		log.Fatal("Could not create table: %s", err.Error())
	}

	st := &store{
		db:  db,
		cmd: make(chan []byte),
	}

	go st.run()
	go h.run(st)

	// more crappy metric handling!
	http.HandleFunc("/metrics", func(resp http.ResponseWriter,
		req *http.Request) {
		Log("Metric request...")
		st.cmd <- []byte("s")
		enc_use := <-st.cmd
		if enc_use == nil {
			enc_use = []byte("{}")
		}
		use := usage{}
		json.Unmarshal(enc_use, &use)
		rep, _ := json.Marshal(
			struct {
				Connections int    `json:"number_connections"`
				Age         string `json:"server_age"`
				Use         usage  `json:"usage"`
			}{
				Connections: h.Count(),
				Age:         time.Now().UTC().Sub(born).String(),
				Use:         use,
			})
		resp.Write(rep)
	})

	http.HandleFunc("/", wsHandler)

	http.HandleFunc("/reset", func(resp http.ResponseWriter, req *http.Request) {
	    Log("Reset...")
	    st.cmd <- []byte("r")
	    info := <-st.cmd
	    reset_info := reset_response{}
	    json.Unmarshal(info, &reset_info)
	    resp.Write([]byte(fmt.Sprintf("Clock reset to 0, we were up to %d minutes...", reset_info.Previous)))
	})

	log.Printf("Starting up server at %s\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("Could not start server:", err)
	}

}
