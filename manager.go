package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

type Manager struct {
	mu           sync.Mutex
	numOfRooms   int
	rooms        map[string]*Room
	sessionStore *sessions.FilesystemStore
	cancel       context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{
		numOfRooms:   0,
		rooms:        make(map[string]*Room),
		sessionStore: sessions.NewFilesystemStore("", []byte("secret-key")),
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
	readBufferSize  = 2048
	writeBufferSize = 2048
)

var upgrader = &websocket.Upgrader{ReadBufferSize: readBufferSize, WriteBufferSize: writeBufferSize}
var codeChars string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+-!~$#&"

func generateRandomCode(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	codeRunes := []rune(codeChars)
	code := make([]rune, length)

	for i := 0; i < length; i++ {
		code[i] = codeRunes[rand.Intn(len(codeRunes))]
	}

	return string(code)
}

func clientNameChecker(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Regular expression to match English letters, numbers, and Chinese characters
		validPattern := regexp.MustCompile(`^[\p{L}0-9]+$`)

		var clientNameCheckerError string
		clientName := r.PostFormValue("clientName")
		fmt.Println(clientName)
		if clientName == "" {
			clientNameCheckerError = "玩家姓名不得為空!"
		} else if len([]rune(clientName)) > 7 {
			clientNameCheckerError = "玩家姓名不得超過七個字元!"
		} else if !validPattern.MatchString(clientName) {
			// Check if the clientName matches the valid pattern
			clientNameCheckerError = "玩家姓名含有非法字元!"
		}

		if clientNameCheckerError != "" {
			// 取得上一頁網址 並捨棄query parameters
			path := r.Header.Get("Referer")
			if path == "" {
				path = "/"
			} else {
				path = stripQueryParameters(r, path)
			}
			// 導回上一頁
			redirectURL := fmt.Sprintf("%s?clientNameCheckerError=%s", path, url.QueryEscape(clientNameCheckerError))
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (m *Manager) joinRoomChecker(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		roomID := req.PostFormValue("roomID")

		clientIP := req.PostFormValue("clientIP")

		clientName := req.PostFormValue("clientName")

		fmt.Println("等待驗證", roomID, clientIP, clientName)

		var invalidRoomID, invalidRoute, invalidJoin, invalidNumOfPlayer, invalidClientIP, invalidClientName string

		if _, ok := m.rooms[roomID]; !ok {
			invalidRoomID = "請輸入存在的房號!"
			redirectURL := fmt.Sprintf("/?invalidRoomID=%s", url.QueryEscape(invalidRoomID))
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}

		r := m.rooms[roomID]

		if r.isInvited {
			invalidRoute = "該房間只透過邀請網址加入!"
			redirectURL := fmt.Sprintf("/?invalidRoute=%s", url.QueryEscape(invalidRoute))
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}

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

		data := &ContextData{
			roomID:     roomID,
			clientIP:   clientIP,
			clientName: clientName,
		}

		/*
			用context去傳資料比較適合用於middleware跟endpoint的對接，如果想用在redirect之類的話是沒辦法的
			因為做redirect的時候，browser會自動給一個request，導致WithValue context不能保存
		*/

		ctx := context.WithValue(req.Context(), dataKey, data)
		newReq := req.WithContext(ctx)

		next(w, newReq)
	}
}

func (m *Manager) home(w http.ResponseWriter, r *http.Request) {
	clientNameCheckerError := r.URL.Query().Get("clientNameCheckerError")
	invalidRoomID := r.URL.Query().Get("invalidRoomID")
	invalidLink := r.URL.Query().Get("invalidLink")
	isInvited := r.URL.Query().Get("invalidRoute")
	invalidClientIP := r.URL.Query().Get("invalidClientIP")
	invalidClientName := r.URL.Query().Get("invalidClientName")
	invalidJoin := r.URL.Query().Get("invalidJoin")
	invalidNumOfPlayer := r.URL.Query().Get("invalidNumOfPlayer")
	invalidOtpID := r.URL.Query().Get("invalidOtpID")
	unknownError := r.URL.Query().Get("unknownError")

	data := map[string]string{
		"clientNameCheckerError": clientNameCheckerError,
		"invalidRoomID":          invalidRoomID,
		"invalidLink":            invalidLink,
		"isInvited":              isInvited,
		"invalidClientIP":        invalidClientIP,
		"invalidClientName":      invalidClientName,
		"invalidJoin":            invalidJoin,
		"invalidNumOfPlayer":     invalidNumOfPlayer,
		"invalidOtpID":           invalidOtpID,
		"unknownError":           unknownError,
	}

	homeTmp.Execute(w, data)
}

type EnterData struct {
	RoomID          string // 讓每個成員都是exported，這樣template才吃的到
	InviteLink      string
	ClientIP        string
	ClientName      string
	MaxNumOfClients []int
	BaseURL         string
}

type ContextData struct {
	roomID     string
	clientIP   string
	clientName string
}

type customedKey struct{} // key值通常是要自訂的type比較保險，所以就隨便設一個type

var dataKey = customedKey{} // 用來當作withvalue context的唯一key值

// PRG pattern for enter a room
func (m *Manager) enter(w http.ResponseWriter, req *http.Request) {
	fmt.Println("enter")
	var invalidRoomID, invalidsessionID string

	cookie, err := req.Cookie("sessionID")
	if err != nil {
		invalidsessionID = "憑證錯誤或過期!"
		redirectURL := fmt.Sprintf("/?invalidOtpID=%s", url.QueryEscape(invalidsessionID))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}
	sessionID := cookie.Value
	fmt.Println("sessionID:", sessionID)

	var session *sessions.Session
	if session, _ = m.sessionStore.Get(req, sessionID); session.IsNew {
		invalidsessionID = "憑證錯誤或過期!"
		redirectURL := fmt.Sprintf("/?invalidOtpID=%s", url.QueryEscape(invalidsessionID))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}
	roomID := session.Values["roomID"].(string)
	clientIP := session.Values["clientIP"].(string)
	clientName := session.Values["clientName"].(string)

	if _, ok := m.rooms[roomID]; !ok {
		invalidRoomID = "請輸入存在的房號!"
		redirectURL := fmt.Sprintf("/?invalidRoomID=%s", url.QueryEscape(invalidRoomID))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	r := m.rooms[roomID]

	// Set MaxAge to -1 to delete the session cookie
	session.Options.MaxAge = -1
	err = session.Save(req, w)
	if err != nil {
		http.Error(w, "server error when deleting session", http.StatusInternalServerError)
		return
	}

	fmt.Println(roomID, clientIP, sessionID, clientName, r.maxNumOfClients)

	encodedRoomID := base64.StdEncoding.EncodeToString([]byte(roomID))
	enterData := EnterData{
		RoomID:          roomID,
		InviteLink:      "https://" + req.Host + "/invite/" + encodedRoomID,
		ClientIP:        clientIP,
		ClientName:      clientName,
		MaxNumOfClients: make([]int, r.maxNumOfClients),
		BaseURL:         req.Host,
	}

	drawTmp.Execute(w, enterData)
}

// 在大廳創建房間
func (m *Manager) postCreateRoom(w http.ResponseWriter, req *http.Request) {
	var roomID string
	for {
		roomID = generateRandomCode(8)
		if _, ok := m.rooms[roomID]; !ok {
			break
		}
	}

	fmt.Println("new roomID: ", roomID)
	clientIP := req.PostFormValue("clientIP")
	clientName := req.PostFormValue("clientName")
	fmt.Println(roomID, clientIP, clientName)

	ctx, cancel := context.WithCancel(context.Background())
	r := newRoom(roomID, ctx, cancel)
	m.rooms[roomID] = r
	m.numOfRooms += 1
	fmt.Println("number of rooms: ", m.numOfRooms)

	// Generate a session ID and create a session
	var sessionID string
	var session *sessions.Session
	var err error
	m.mu.Lock()
	for {
		sessionID = generateRandomCode(8)
		session, err = m.sessionStore.Get(req, sessionID)
		if err != nil {
			fmt.Println("error when getting session", err)
		}
		if session.IsNew {
			session.Values["roomID"] = roomID
			session.Values["clientIP"] = clientIP
			session.Values["clientName"] = clientName
			break
		}
	}
	m.mu.Unlock()
	fmt.Println("unlocked")
	// Save the session
	err = session.Save(req, w)
	if err != nil {
		http.Error(w, "server error when saving client session", http.StatusInternalServerError)
		return
	}

	fmt.Println("sessionID:", sessionID)
	cookie := http.Cookie{
		Name:     "sessionID",
		Value:    sessionID,
		HttpOnly: true,
		MaxAge:   60,
	}
	http.SetCookie(w, &cookie)

	go r.run()

	// Redirect to a GET request with the roomID, clientIP, clientName as a query parameter
	nextURL := "/draw"
	http.Redirect(w, req, nextURL, http.StatusSeeOther)

}

// 透過在大廳輸入房號加入他人房間
func (m *Manager) postRoomIDJoin(w http.ResponseWriter, req *http.Request) {
	data, ok := req.Context().Value(dataKey).(*ContextData)
	if !ok {
		log.Println("error when getting data from context")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	roomID := data.roomID
	clientIP := data.clientIP
	clientName := data.clientName

	r := m.rooms[roomID]

	// Generate a session ID and create a session
	var sessionID string
	var session *sessions.Session
	var err error
	m.mu.Lock()
	for {
		sessionID = generateRandomCode(8)
		session, err = m.sessionStore.Get(req, sessionID)
		if err != nil {
			fmt.Println("error when getting session", err)
		}
		if session.IsNew {
			session.Values["roomID"] = roomID
			session.Values["clientIP"] = clientIP
			session.Values["clientName"] = clientName
			break
		}
	}
	m.mu.Unlock()
	fmt.Println("unlocked")
	// Save the session
	err = session.Save(req, w)
	if err != nil {
		http.Error(w, "server error when saving client session", http.StatusInternalServerError)
		return
	}

	fmt.Println("sessionID:", sessionID)
	cookie := http.Cookie{
		Name:     "sessionID",
		Value:    sessionID,
		HttpOnly: true,
		MaxAge:   60,
	}
	http.SetCookie(w, &cookie)

	go r.run()

	// Redirect to a GET request with the roomID, clientIP, clientName as a query parameter
	nextURL := "/draw"
	http.Redirect(w, req, nextURL, http.StatusSeeOther)
}

// 要把所有query parameters去掉可以用下列方法
func stripQueryParameters(r *http.Request, path string) string {
	if path == "" {
		path = r.URL.Path
	}
	pathObject, _ := url.Parse(path)
	return pathObject.Scheme + "://" + pathObject.Host + pathObject.Path
}

// 邀請連結畫面
func (m *Manager) getInviteJoin(w http.ResponseWriter, req *http.Request) {
	var clientNameCheckerError, invalidClientIP, invalidJoin, invalidNumOfPlayer, invalidClientName string
	clientNameCheckerError = req.URL.Query().Get("clientNameCheckerError")
	invalidClientName = req.URL.Query().Get("invalidClientName")
	invalidClientIP = req.URL.Query().Get("invalidClientIP")
	invalidJoin = req.URL.Query().Get("invalidJoin")
	invalidNumOfPlayer = req.URL.Query().Get("invalidNumOfPlayer")
	encodedRoomID := mux.Vars(req)["encodedRoomID"]
	fmt.Println(encodedRoomID)

	// 傳送encodedRoomID給template，並隱藏在hidden input裡面
	data := map[string]string{
		"clientNameCheckerError": clientNameCheckerError,
		"invalidClientName":      invalidClientName,
		"invalidClientIP":        invalidClientIP,
		"invalidJoin":            invalidJoin,
		"invalidNumOfPlayer":     invalidNumOfPlayer,
		"encodedRoomID":          encodedRoomID,
	}
	inviteTmp.Execute(w, data)
}

// 透過邀請連結加入他人房間
func (m *Manager) postInviteJoin(w http.ResponseWriter, req *http.Request) {
	var invalidClientIP, invalidClientName, invalidLink, invalidJoin, invalidNumOfPlayer string
	encodedRoomID := req.PostFormValue("encodedRoomID")
	roomID, err := base64.StdEncoding.DecodeString(encodedRoomID) // type []byte

	clientIP := req.PostFormValue("clientIP")

	clientName := req.PostFormValue("clientName")

	fmt.Println("等待驗證", roomID, clientIP, clientName)

	// 解密roomID
	if _, ok := m.rooms[string(roomID)]; !ok || err != nil {
		invalidLink = "無效連結或已失效!"
		redirectURL := fmt.Sprintf("/?invalidLink=%s", url.QueryEscape(invalidLink))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	r := m.rooms[string(roomID)]

	// 取得baseURL
	//path := "https://" + req.Host

	// 取得上一頁網址 並捨棄query parameters
	path := req.Header.Get("Referer")
	if path == "" {
		path = "/"
	} else {
		path = stripQueryParameters(req, path)
	}

	if r.isPlaying {
		invalidJoin = "該房間已開始遊戲!"
		redirectURL := fmt.Sprintf("%s?invalidJoin=%s", path, url.QueryEscape(invalidJoin))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	if r.numOfClients >= r.maxNumOfClients {
		invalidNumOfPlayer = "該房間人數已達上限!"
		redirectURL := fmt.Sprintf("%s?invalidNumOfPlayer=%s", path, url.QueryEscape(invalidNumOfPlayer))
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	for _, client := range r.clients {
		if clientIP == client.ip {
			invalidClientIP = "請勿重複加入該房間!"
			redirectURL := fmt.Sprintf("%s?invalidClientIP=%s", path, url.QueryEscape(invalidClientIP))
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}
		if clientName == client.name {
			invalidClientName = "已有同名玩家在該房間!"
			redirectURL := fmt.Sprintf("%s?invalidClientName=%s", path, url.QueryEscape(invalidClientName))
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}
	}

	fmt.Println("驗證成功", roomID, clientIP, clientName)

	// Generate a session ID and create a session
	var sessionID string
	var session *sessions.Session
	m.mu.Lock()
	for {
		sessionID = generateRandomCode(8)
		session, err = m.sessionStore.Get(req, sessionID)
		if err != nil {
			fmt.Println("error when getting session", err)
		}
		if session.IsNew {
			session.Values["roomID"] = string(roomID)
			session.Values["clientIP"] = clientIP
			session.Values["clientName"] = clientName
			break
		}
	}
	m.mu.Unlock()
	fmt.Println("unlocked")
	// Save the session
	err = session.Save(req, w)
	if err != nil {
		http.Error(w, "server error when saving client session", http.StatusInternalServerError)
		return
	}

	fmt.Println("sessionID:", sessionID)
	cookie := http.Cookie{
		Name:     "sessionID",
		Value:    sessionID,
		HttpOnly: true,
		MaxAge:   60,
	}
	http.SetCookie(w, &cookie)

	redirectURL := "/draw"
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
