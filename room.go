package main

import (
	"context"
	"fmt"
	"sync"
)

type Room struct {
	roomID       string
	numOfClients int
	clients      map[*Client]bool
	join         chan *Client
	leave        chan *Client
	ChatArea     chan *Msg
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

type Data struct {
	Content   string `json:"content"`
	SysOrChat bool   `json:"sysOrChat"`
}

type Msg struct {
	Type    string `json:"type"`
	Payload Data   `json:"payload"`
}

func newRoom(id string, ctx context.Context, cancel context.CancelFunc) *Room {
	return &Room{
		roomID:       id,
		numOfClients: 0,
		clients:      make(map[*Client]bool),
		join:         make(chan *Client),
		leave:        make(chan *Client),
		ChatArea:     make(chan *Msg, 10),
		mu:           sync.RWMutex{},
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (r *Room) run(ctx context.Context) {
	defer func() {
		r.mu.RLock()
		fmt.Println("room close")
		close(r.join)
		close(r.leave)
		close(r.ChatArea)
		r.mu.RUnlock()
	}()
	for {
		select {
		case client := <-r.join:
			r.mu.Lock()
			r.clients[client] = true
			// r.ChatArea <- []byte("$new join") // 如果我們要在同一個select case去觸發其他case(此例是對channel傳訊息)，就要把這個channel變成buffer channel，這樣才不會導致這則訊息塞住，而其他的訊息進不來造成訊息卡死
			r.ChatArea <- &Msg{Type: "text", Payload: Data{Content: "new join", SysOrChat: true}}
			r.mu.Unlock()
		case client := <-r.leave:
			r.mu.Lock()
			delete(r.clients, client)
			r.numOfClients--
			r.mu.Unlock()
			close(client.receive)
		case msg := <-r.ChatArea:
			r.mu.RLock()
			for client := range r.clients {
				client.receive <- &Msg{Type: msg.Type, Payload: msg.Payload}
			}
			r.mu.RUnlock()
		case <-ctx.Done():
			fmt.Printf("room %s is empty\n", r.roomID)
			return
		}
	}
}

/*
type Room struct {
	roomID string

	numOfClients int

	clients map[*Client]bool

	join chan *Client

	leave chan *Client

	ChatArea chan []byte
}

func newRoom(id string) *Room {
	return &Room{
		roomID:       id,
		numOfClients: 0,
		clients:      make(map[*Client]bool),
		join:         make(chan *Client),
		leave:        make(chan *Client),
		ChatArea:     make(chan []byte),
	}
}

func (r *Room) run(manager *Manager) {
outerloop:
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
		case client := <-r.leave:
			r.numOfClients -= 1
			delete(r.clients, client)
			close(client.receive)
			if r.numOfClients == 0 {
				fmt.Printf("room %s is empty\n", r.roomID)
				manager.isEmpty <- r.roomID
				break outerloop
			}
		case msg := <-r.ChatArea:
			for client := range r.clients {
				client.receive <- msg
			}
		}
	}
}

/*
type Room struct {
	roomID string

	manager *Manager

	numOfClients int

	clients map[*Client]bool

	join chan *Client

	leave chan *Client

	ChatArea chan []byte
}

func newRoom(id string) *Room {
	return &Room{
		roomID:       id,
		numOfClients: 0,
		clients:      make(map[*Client]bool),
		join:         make(chan *Client),
		leave:        make(chan *Client),
		ChatArea:     make(chan []byte),
	}
}

func (r *Room) run(manager *Manager) {

outerloop:
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
		case client := <-r.leave:
			r.numOfClients -= 1
			delete(r.clients, client)
			close(client.receive)
			if r.numOfClients == 0 {
				fmt.Printf("room %s is empty\n", r.roomID)
				manager.isEmpty <- r.roomID
				break outerloop
			}
		case msg := <-r.ChatArea:
			for client := range r.clients {
				client.receive <- msg
			}
		}
	}
}

/*
func (r *Room) serverWS(w http.ResponseWriter, req *http.Request) {
	fmt.Println(mux.Vars(req)["roomID"])
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Println("Upgrade ERROR: ", err)
	}
	client := &Client{
		con:     socket,
		receive: make(chan []byte),
		room:    r,
	}

	r.join <- client
	client.room.ChatArea <- []byte("new join")
	defer func() {
		client.room.ChatArea <- []byte("someone is leaving")
		r.leave <- client
	}()
	// 這裡要注意這兩行的寫法，相反會在關閉的時候出錯
	go client.WriteMsg()
	client.readMsg()

}
*/
