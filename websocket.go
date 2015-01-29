package websocks

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	writeWait  = 10 * time.Second
	readWait   = 10 * time.Second
	pingPeriod = (readWait * 9) / 10
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// Conn type will have outbound and inbound channels, and
// the websocket Conn.
type Conn struct {
	ws            *websocket.Conn
	Outbound      chan []byte
	Inbound       chan []byte
	EventHandlers map[string][]socksEventHandler
}

type Msg struct {
	Type    string `json:"type"`
	Message string `json:"msg"`
}

// reader is started as a routine, it will continue to read data from
// websocket Conn and sends it to the Conns inbound channel
// as strings
func (c *Conn) reader() {
	log.Print("Conn reader gorouting starting.")
	defer func() {
		log.Print("Conn reader gorouting stopping.")
		close(c.Inbound)
		c.ws.Close()
	}()
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(writeWait))
		return nil
	})
	c.ws.SetReadDeadline(time.Now().Add(readWait))
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		var m Msg
		if err := json.Unmarshal(message, &m); err != nil {
			log.Println(err.Error())
		}
		c.callHandler(m)
	}
}

// Write message as byte array to Conn, with messagetype
func (c *Conn) write(mt int, payload []byte) error {
	//log.Print("Conn.write() called")
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// Routine to continue to write from outbound channel to websocket
// Conn. Will close outbound channel when closed.
func (c *Conn) writer() {
	log.Print("Conn writer gorouting starting.")
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		log.Print("Conn writer gorouting stopping.")
		pingTicker.Stop()
		close(c.Outbound)
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.Outbound:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				log.Println("[Conn.writePump] !ok.")
				return
			}
			if err := c.write(websocket.TextMessage, []byte(message)); err != nil {
				log.Println("[Conn.writePump] err: '", err, "'.")
				c.callHandler(Msg{Type: "disconnect"})
				return
			}
		// When pingTicker ticks, send a PingMessage to client.
		case <-pingTicker.C:
			if err := c.write(websocket.PingMessage, []byte("{\"type\":\"ping\"}")); err != nil {
				log.Println("[Conn.writePump] pingTicker err: '", err, "'.")
				log.Println("ping")
				return
			}
		}
	}
}

type socksEventHandler func(Msg)

func (c *Conn) On(msgType string, f socksEventHandler) {
	if c.EventHandlers[msgType] == nil {
		c.EventHandlers[msgType] = make([]socksEventHandler, 0)
	}
	c.EventHandlers[msgType] = append(c.EventHandlers[msgType], f)
}

func (c *Conn) Send(msg Msg) {
	b, err := json.Marshal(&msg)
	if err != nil {
		log.Println(err.Error())
		return
	}
	c.Outbound <- b
}

func (c *Conn) callHandler(msg Msg) {
	log.Println("got msg.type", msg.Type)
	if c.EventHandlers[msg.Type] != nil {
		for _, f := range c.EventHandlers[msg.Type] {
			f(msg)
			log.Println("called handler for msg.type", msg.Type)
		}
	}
}
