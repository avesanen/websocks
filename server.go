package websocks

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zenazn/goji/web"
)

type Server struct {
	*sync.RWMutex
	ConnectHandler func(*Conn)
	Conns          []*Conn
}

func NewServer() *Server {
	s := &Server{}
	s.RWMutex = &sync.RWMutex{}
	return s
}

func (s *Server) WebsocketHandler(webC web.C, w http.ResponseWriter, r *http.Request) {
	id := webC.URLParams["id"]
	log.Printf("New websocket request on id %s.", id)

	// Only get requests
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Force same origin policy
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Origin not allowed", 403)
		return
	}

	// Try to init websocket Conn
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			http.Error(w, "Not a websocket handshake", 400)
		}
		return
	}

	// Wrap the websocket into a Conn and start reader/writer routines
	c := &Conn{}
	c.ws = ws
	c.Id = id
	c.EventHandlers = make(map[string][]socksEventHandler)
	c.Outbound = make(chan []byte)
	c.Inbound = make(chan []byte)
	go c.reader()
	go c.writer()

	if s.ConnectHandler != nil {
		s.ConnectHandler(c)
	}

	s.Lock()
	defer s.Unlock()
	s.Conns = append(s.Conns, c)
	c.On("disconnect", func(msg Msg) {
		for i, k := range s.Conns {
			if k == c {
				s.Conns = append(s.Conns[:i], s.Conns[i+1:]...)
			}
		}
	})
}

func (s *Server) OnConnect(f func(*Conn)) {
	s.ConnectHandler = f
}

func (s *Server) BroadCast() {}
