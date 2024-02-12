package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Manager struct {
	mu         sync.Mutex
	numOfRooms int
	rooms      map[string]*Room
	cancel     context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{
		numOfRooms: 0,
		rooms:      make(map[string]*Room),
	}
}

func (m *Manager) run(ctx context.Context) {

	defer func() {
		m.mu.Lock()
		fmt.Println("Manager is stop running")
		m.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Manager closed")
			return
		}
	}
}

const (
	readBufferSize  = 1024
	writeBufferSize = 1024
)

var upgrader = &websocket.Upgrader{ReadBufferSize: readBufferSize, WriteBufferSize: writeBufferSize}

func (m *Manager) serverWS(w http.ResponseWriter, req *http.Request) {
	roomID := mux.Vars(req)["roomID"]
	fmt.Println(roomID)
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Println("Upgrade ERROR: ", err)
	}
	fmt.Println("before lock")
	m.mu.Lock()
	fmt.Println("enter lock")
	if _, ok := m.rooms[roomID]; !ok {
		ctx, cancel := context.WithCancel(context.Background())
		m.numOfRooms += 1
		fmt.Println("room number: ", m.numOfRooms)
		m.rooms[roomID] = newRoom(roomID, ctx, cancel)
		r := m.rooms[roomID]
		go r.run(r.ctx)
	}
	fmt.Println(m.rooms)
	r := m.rooms[roomID]
	r.numOfClients += 1
	client := &Client{
		rL:      sync.RWMutex{},
		wL:      sync.RWMutex{},
		con:     socket,
		receive: make(chan *Msg),
		room:    r,
	}
	r.join <- client // 這行的邏輯之後會增加一個系統聊天頻道，共用同個ws connect，但接收方會分流
	//r.ChatArea <- []byte("new join") 在select外面觸發比較保險
	fmt.Println("end lock")
	m.mu.Unlock()

	defer func() {
		r.ChatArea <- &Msg{Type: "text", Payload: Data{Content: "someone leaves", SysOrChat: true}}
		time.Sleep(500 * time.Millisecond)
		r.leave <- client
		fmt.Println("connection close")
		fmt.Println("client close")
		client.con.Close() // 離開記得關閉socket
		if r.numOfClients == 0 {
			r.cancel()
			m.numOfRooms -= 1
			delete(m.rooms, r.roomID)
		}
		fmt.Println("number of rooms: ", m.numOfRooms)
		if m.numOfRooms == 0 {
			m.cancel()
		}

	}()
	fmt.Println("go write")
	go client.WriteMsg()
	go client.readMsg()
	go client.WriteMsg()
	client.readMsg()
}

/*
type joinMsg struct {
	roomID string
	client *Client
	msg    []byte
}

type leaveMsg struct {
	roomID string
	client *Client
	msg    []byte
}

type Manager struct {
	numOfRooms int
	isEmpty    chan string
	rooms      map[string]*Room
	join       chan *joinMsg
	leave      chan *leaveMsg
}

func newManager() *Manager {
	return &Manager{
		numOfRooms: 0,
		isEmpty:    make(chan string),
		rooms:      make(map[string]*Room),
		join:       make(chan *joinMsg),
		leave:      make(chan *leaveMsg),
	}
}

func (m *Manager) run() {
	for {
		select {
		case joinMsg := <-m.join:
			m.rooms[joinMsg.roomID].join <- joinMsg.client
			m.rooms[joinMsg.roomID].ChatArea <- []byte("new join")
		case leaveMsg := <-m.leave:
			m.rooms[leaveMsg.roomID].leave <- leaveMsg.client
			m.rooms[leaveMsg.roomID].ChatArea <- []byte("someone is leaving")
		case roomID := <-m.isEmpty:
			m.numOfRooms -= 1
			r := m.rooms[roomID]
			close(r.join)
			close(r.leave)
			close(r.ChatArea)
			delete(m.rooms, r.roomID)
			if m.numOfRooms == 0 {
				fmt.Println("All Rooms Are EMPTY")
			}
		}

	}
}

const (
	readBufferSize  = 1024
	writeBufferSize = 1024
)

var upgrader = &websocket.Upgrader{ReadBufferSize: readBufferSize, WriteBufferSize: writeBufferSize}

func (m *Manager) serverWS(w http.ResponseWriter, req *http.Request) {
	roomID := mux.Vars(req)["roomID"]
	fmt.Println(roomID)
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Println("Upgrade ERROR: ", err)
	}
	if _, ok := m.rooms[roomID]; !ok {
		m.numOfRooms += 1
		m.rooms[roomID] = newRoom(roomID)
		r := m.rooms[roomID]
		go r.run(m)
	}
	fmt.Println(m.rooms)
	r := m.rooms[roomID]
	r.manager = m
	r.numOfClients += 1
	client := &Client{
		con:     socket,
		receive: make(chan []byte),
		room:    r,
	}
	//r.ChatArea <- []byte("new join")
	joinMsg := &joinMsg{roomID: roomID, client: client, msg: []byte("new join")}
	m.join <- joinMsg
	defer func() {
		//r.ChatArea <- []byte("someone is leaving")
		leaveMsg := &leaveMsg{roomID: roomID, client: client, msg: []byte("someone is leaving")}
		m.leave <- leaveMsg
		client.con.Close()
	}()
	// 這裡要注意這兩行的寫法，相反會在關閉的時候出錯
	go client.WriteMsg()
	client.readMsg()

}
*/
