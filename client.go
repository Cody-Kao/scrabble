package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	rL sync.RWMutex

	wL sync.RWMutex

	con *websocket.Conn

	receive chan *Msg

	room *Room
}

func (c *Client) readMsg() {
	for {
		c.rL.Lock()
		_, receivedMsg, err := c.con.ReadMessage()
		c.rL.Unlock()

		if err != nil {
			fmt.Println("readMsg ERROR: ", err)
			break // 因為在connection close之後如果不break，我們會繼續對已關閉的WS進行ReadMessage導致出錯
		}
		var Msg Msg

		// same as: err = json.NewDecoder(bytes.NewReader(b.Bytes())).Decode(&receivedMsg)
		err = json.NewDecoder(bytes.NewReader(receivedMsg)).Decode(&Msg)
		if err != nil {
			log.Fatal("Error decoding JSON:", err)
			return
		}
		fmt.Println("read: ", Msg)

		c.room.ChatArea <- &Msg

	}
}

func (c *Client) WriteMsg() {
	// 這裡送出從json格式的Msg
	for forwardMsg := range c.receive {
		fmt.Println("write: ", string(forwardMsg.Payload.Content))
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&forwardMsg)
		if err != nil {
			log.Fatal("Error encoding to JSON:", err)
			return
		}
		fmt.Println(b)
		c.wL.Lock()
		err = c.con.WriteMessage(websocket.TextMessage, b.Bytes())
		c.wL.Unlock()
		if err != nil {
			fmt.Println("WriteMsg ERROR: ", err)
			break
		}
	}
}
