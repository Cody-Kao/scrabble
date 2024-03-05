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
	rL *sync.RWMutex

	wL *sync.RWMutex

	ip string

	name string

	isLeft bool

	score int

	con *websocket.Conn

	receive chan *Msg

	receiveScoreUpdate chan *ScoreDict

	room *Room

	mu *sync.RWMutex
}

func newClient(clientIP, clientName string, socket *websocket.Conn, r *Room) *Client {
	return &Client{
		rL: &sync.RWMutex{},
		wL: &sync.RWMutex{},

		ip:                 clientIP,
		name:               clientName,
		isLeft:             false,
		score:              0,
		con:                socket,
		receive:            make(chan *Msg, 100),
		receiveScoreUpdate: make(chan *ScoreDict, 100),
		room:               r,
		mu:                 &sync.RWMutex{},
	}
}

func (c *Client) readMsg(wg *sync.WaitGroup) {
	// 在這裡呼叫write message、SendUpdateScore，這樣就可以更好的控制該function的開關
	go c.SendMsg()
	go c.SendUpdateScore()
	defer func() {
		close(c.receive)
		close(c.receiveScoreUpdate)
		wg.Done()
	}()

	for {
		c.rL.RLock()
		_, receivedMsg, err := c.con.ReadMessage()
		c.rL.RUnlock()

		if err != nil {
			fmt.Println("readMsg ERROR: ", err)
			break // 因為在connection close之後如果不break，我們會繼續對已關閉的WS進行ReadMessage導致出錯
		}
		var Msg Msg
		fmt.Println(receivedMsg)
		err = json.NewDecoder(bytes.NewReader(receivedMsg)).Decode(&Msg)
		fmt.Println("read: ", Msg)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
			return
		}

		if Msg.Type == "IN" || Msg.Type == "GS" || Msg.Type == "CS" || Msg.Type == "RO" || Msg.Type == "sys" || Msg.Type == "RSK" {
			c.room.gameControlChan <- &Msg
			continue
		}

		c.room.BroadcastArea <- &Msg
	}
}

func (c *Client) SendMsg() {
	for forwardMsg := range c.receive {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&forwardMsg)
		if err != nil {
			log.Fatal("Error encoding to JSON:", err)
			return
		}

		c.wL.Lock()
		err = c.con.WriteMessage(websocket.TextMessage, b.Bytes())
		c.wL.Unlock()
		if err != nil {
			fmt.Println("SendMsg ERROR: ", err)
			break
		}
		fmt.Println("data send out: ", b.Bytes())
	}
}

func (c *Client) SendUpdateScore() {
	for score := range c.receiveScoreUpdate {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&score)
		if err != nil {
			log.Fatal("Error encoding to JSON:", err)
			return
		}
		fmt.Println(b)
		c.wL.Lock()
		err = c.con.WriteMessage(websocket.TextMessage, b.Bytes())
		c.wL.Unlock()
		if err != nil {
			fmt.Println("SendUpdateScore ERROR: ", err)
			break
		}
	}
}
