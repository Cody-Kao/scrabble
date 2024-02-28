package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
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
var codeChars string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!"

func generateRandomCode(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	codeRunes := []rune(codeChars)
	code := make([]rune, length)

	for i := 0; i < length; i++ {
		code[i] = codeRunes[rand.Intn(len(codeRunes))]
	}

	return string(code)
}

/* 這是直接透過一個邀請網址加入的，難點在於需要建立使用者名稱以及取得使用者電腦IP
func (m *Manager) getJoin(w http.ResponseWriter, req *http.Request) {
	roomID := mux.Vars(req)["roomID"]
	if _, ok := m.rooms[roomID]; !ok {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	clientIP := "123456"
	clientName := "test"
	m.mu.Lock()
	fmt.Println("enter lock")
	r := m.rooms[roomID]
	fmt.Println(m.rooms)
	client := &Client{
		//rL:      sync.RWMutex{},
		//wL:      sync.RWMutex{},
		ip:      clientIP,
		name:    clientName,
		receive: make(chan *Msg),
		room:    r,
	}
	r.join <- client
	fmt.Println("end lock")
	m.mu.Unlock()
	fmt.Println("roomID: ", roomID)
	fmt.Println("room number: ", m.numOfRooms)
	data := map[string]string{
		"roomID":     roomID,
		"clientIP":   clientIP,
		"clientName": clientName,
	}
	drawTmp.Execute(w, data)
}
*/

func (m *Manager) home(w http.ResponseWriter, r *http.Request) {
	invalidRoomID := r.URL.Query().Get("invalidRoomID")
	invalidClientIP := r.URL.Query().Get("invalidClientIP")
	invalidClientName := r.URL.Query().Get("invalidClientName")
	data := map[string]string{
		"invalidRoomID":     invalidRoomID,
		"invalidClientIP":   invalidClientIP,
		"invalidClientName": invalidClientName,
	}

	homeTmp.Execute(w, data)
}

// PRG pattern
func (m *Manager) enter(w http.ResponseWriter, req *http.Request) {
	fmt.Println("enter")
	roomID := req.URL.Query().Get("roomID")
	clientIP := req.URL.Query().Get("clientIP")
	clientName := req.URL.Query().Get("clientName")
	invalidJoin := req.URL.Query().Get("invalidJoin")
	invalidNumOfPlayer := req.URL.Query().Get("invalidNumOfPlayer")
	fmt.Println(roomID, clientIP, clientName, invalidJoin, invalidNumOfPlayer)

	data := map[string]string{
		"roomID":             roomID,
		"clientIP":           clientIP,
		"clientName":         clientName,
		"invalidJoin":        invalidJoin,
		"invalidNumOfPlayer": invalidNumOfPlayer,
	}

	drawTmp.Execute(w, data)
}

// 在大廳創建房間
func (m *Manager) postCreateRoom(w http.ResponseWriter, req *http.Request) {
	var roomID string
	for {
		roomID = generateRandomCode(6)
		if _, ok := m.rooms[roomID]; !ok {
			break
		}
	}

	fmt.Println("new roomID: ", roomID)
	clientIP := req.PostFormValue("clientIP")
	/*
		clientIP, port, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			// Handle the error
			fmt.Println("Error getting client IP:", err)
			return
		}
	*/
	clientName := req.PostFormValue("clientName")
	fmt.Println(roomID, clientIP, clientName)

	ctx, cancel := context.WithCancel(context.Background())
	r := newRoom(roomID, ctx, cancel)
	m.rooms[roomID] = r
	m.numOfRooms += 1
	fmt.Println("number of rooms: ", m.numOfRooms)
	go r.run()

	// Redirect to a GET request with the roomID, clientIP, clientName as a query parameter
	redirectURL := fmt.Sprintf("/draw?roomID=%s&clientIP=%s&clientName=%s", roomID, clientIP, clientName)
	http.Redirect(w, req, redirectURL, http.StatusSeeOther)
}

// 透過在大廳輸入房號加入他人房間
func (m *Manager) postRoomIDJoin(w http.ResponseWriter, req *http.Request) {
	var invalidRoomID, invalidClientIP, invalidClientName, invalidJoin, invalidNumOfPlayer string
	roomID := req.PostFormValue("roomID")
	if _, ok := m.rooms[roomID]; !ok {
		invalidRoomID = "請輸入存在的房號!"
		redirectURL := fmt.Sprintf("/?invalidRoomID=%s", url.QueryEscape(invalidRoomID))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	r := m.rooms[roomID]

	if r.isPlaying {
		invalidJoin = "該房間已開始遊戲!"
		redirectURL := fmt.Sprintf("/?invalidJoin=%s", url.QueryEscape(invalidJoin))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	if r.numOfClients >= r.maxNumOfClients {
		invalidNumOfPlayer = "該房間人數已達上限!"
		redirectURL := fmt.Sprintf("/?invalidNumOfPlayer=%s", url.QueryEscape(invalidNumOfPlayer))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	clientIP := req.PostFormValue("clientIP")
	/*
		clientIP, port, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			// Handle the error
			fmt.Println("Error getting client IP:", err)
			return
		}
	*/
	clientName := req.PostFormValue("clientName")
	for _, client := range r.clients {
		if clientIP == client.ip {
			invalidClientIP = "請勿重複加入該房間!"
			redirectURL := fmt.Sprintf("/?invalidClientIP=%s", url.QueryEscape(invalidClientIP))
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}
		if clientName == client.name {
			invalidClientName = "已有同名玩家在該房間!"
			redirectURL := fmt.Sprintf("/?invalidClientName=%s", url.QueryEscape(invalidClientName))
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}
	}
	fmt.Println("驗證成功", roomID, clientIP, clientName)

	// Redirect to a GET request with the roomID, clientIP, clientName as a query parameter
	// 確保特殊字元不會出錯，所以用url.QueryEscape()包覆
	redirectURL := fmt.Sprintf("/draw?roomID=%s&clientIP=%s&clientName=%s",
		url.QueryEscape(roomID), url.QueryEscape(clientIP), url.QueryEscape(clientName))
	http.Redirect(w, req, redirectURL, http.StatusSeeOther)
}

func (m *Manager) serverWS(w http.ResponseWriter, req *http.Request) {
	fmt.Println("servreWS Start")
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Println("Upgrade ERROR: ", err)
	}
	roomID := mux.Vars(req)["roomID"]
	clientIP := req.URL.Query().Get("clientIP")
	clientName := req.URL.Query().Get("clientName")
	fmt.Println("roomID:", roomID, "clientIP:", clientIP, "clientName:", clientName)
	r := m.rooms[roomID]
	client := newClient(clientIP, clientName, socket, r) // add new client
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go client.readMsg(wg)
	m.mu.Lock()
	fmt.Println("enter lock")
	fmt.Println(m.rooms)
	r.clients[clientName] = client
	r.numOfClients += 1
	fmt.Println("end lock")
	m.mu.Unlock()
	r.join <- client
	fmt.Println("number of members:", r.numOfClients)
	fmt.Println("join roomID:", roomID)
	defer func() {
		delete(r.clients, client.name)
		r.numOfClients--
		r.leave <- client
		fmt.Println("connection close")
		fmt.Println("client close")
		client.con.Close() // 離開記得關閉socket
		client.isLeft = true
		fmt.Println("number of members:", r.numOfClients)
		if r.numOfClients == 0 {
			r.cancel()
			m.numOfRooms -= 1
			delete(m.rooms, r.roomID)
		}
		fmt.Println("number of rooms:", m.numOfRooms)
		if m.numOfRooms == 0 {
			m.cancel()
		}
	}()
	fmt.Println("go wait")
	wg.Wait()
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
