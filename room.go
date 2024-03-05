package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type clientNode struct {
	clientName string
	next       *clientNode
	prev       *clientNode
}

type Room struct {
	isClosed        bool
	isPlaying       bool
	isRoundOver     bool
	isGameOver      bool // 如果有人在RO的中途離開，造成GO，就必須讓RS不能在4秒後送出
	isInvited       bool
	curPainterExit  bool // 記住這一輪是否有畫家中途離開
	roomID          string
	answer          string
	numOfClients    int
	winScore        int
	roundScore      int
	maxNumOfClients int
	clients         map[string]*Client
	guessRight      map[string]interface{}
	questionRecord  *map[int]interface{}
	otp             map[string]interface{}
	roomMaster      *clientNode
	curPainter      *clientNode
	join            chan *Client
	leave           chan *Client
	BroadcastArea   chan *Msg
	//scoreUpdateChan chan struct{}
	gameControlChan chan *Msg
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

type ScoreDict struct {
	Type string         `json:"type"`
	Dict map[string]int `json:"dict"`
}

type Data struct {
	Content     string `json:"content,omitempty"` // if a field is empty, it will not be marshalled
	NewPoint    []int  `json:"newPoint,omitempty"`
	StrokeWidth *int   `json:"strokeWidth,omitempty"` // since the zero value of an int isn't really "empty"
	Color       string `json:"color,omitempty"`
	PenStyle    *int   `json:"penStyle,omitempty"` // since the zero value of an int isn't really "empty"
}

// 這個比較通用，因為我們解密的時候需要指名某種struct，而如果無法在解密前就分類訊息類別就只能用同一種struct解密
// 再者，我不太想為了某種訊息就開一個go routine去處理，所以我為了能塞入c.receive chan，就會用這個struct去包裹訊息了
type Msg struct {
	Type    string `json:"type"`
	Payload Data   `json:"payload"`
}

func newRoom(id string, ctx context.Context, cancel context.CancelFunc) *Room {
	return &Room{
		isClosed:        false,
		isPlaying:       false,
		isRoundOver:     false,
		isGameOver:      false,
		isInvited:       false,
		curPainterExit:  false,
		roomID:          id,
		answer:          "",
		numOfClients:    0,
		winScore:        20,
		roundScore:      0,
		maxNumOfClients: 8,
		clients:         make(map[string]*Client),
		guessRight:      make(map[string]interface{}),
		questionRecord:  &map[int]interface{}{},
		otp:             make(map[string]interface{}),
		join:            make(chan *Client, 10),
		leave:           make(chan *Client, 10),
		BroadcastArea:   make(chan *Msg, 100),
		//scoreUpdateChan: make(chan struct{}, 10),
		gameControlChan: make(chan *Msg, 10),
		mu:              sync.RWMutex{},
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (r *Room) RS() {
	if r.isClosed {
		fmt.Println("Room is closed, no more sending RS")
		return
	}
	// 判斷是否已經GO了
	if r.isGameOver {
		fmt.Println("The game is over, no more RS")
		return
	}
	if r.curPainterExit {
		// reset curPaintExiter when a new round begins
		r.curPainterExit = !r.curPainterExit
	} else {
		// set a new curPaint when a new round begins
		r.curPainter = r.curPainter.next
	}
	// use "@" as a delimitor, so the clientName can not contain "@"
	// 記得要順便送出題目選項
	fmt.Println("Start getting questions")
	q1, q2 := getQuestions(&category, r.questionRecord)
	fmt.Println("Stop getting questions")
	r.mu.RLock()
	r.BroadcastArea <- &Msg{Type: "RS", Payload: Data{Content: fmt.Sprintf("%s@%s@%s@%s", r.curPainter.clientName, r.curPainter.next.clientName, q1, q2)}}
	r.mu.RUnlock()
	r.isRoundOver = false
}

func (r *Room) RO(t string) {
	if r.isClosed {
		fmt.Println("Room is closed, no more sending RO")
		return
	}

	// 判斷是否已經GO了
	if r.isGameOver {
		fmt.Println("The game is over, no more RO")
		return
	}

	fmt.Println("Server send RO")
	r.mu.RLock()
	if len(r.guessRight) == 0 {
		r.BroadcastArea <- &Msg{Type: "RO", Payload: Data{Content: "2"}}
	} else {
		r.BroadcastArea <- &Msg{Type: "RO", Payload: Data{Content: t}}
	}
	r.guessRight = map[string]interface{}{}
	r.mu.RUnlock()
	time.Sleep(time.Second * 4)
	r.RS()
}

func (r *Room) RSK() {
	if r.isClosed {
		fmt.Println("Room is closed, no more sending RO")
		return
	}

	// 判斷是否已經GO了
	if r.isGameOver {
		fmt.Println("The game is over, no more RSK")
		return
	}

	fmt.Println("Server send RSK")
	r.mu.RLock()
	r.BroadcastArea <- &Msg{Type: "RSK"}
	r.guessRight = map[string]interface{}{}
	r.mu.RUnlock()
	time.Sleep(time.Second * 4)
	r.RS()
}

func findMax(clients map[string]*Client) int {
	res := 0
	for _, client := range clients {
		if client.score > res {
			res = client.score
		}
	}
	return res
}

func getTimeStamp() (string, error) {
	taipeiLocation, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		fmt.Println("Error loading location:", err)
		return "", err
	}

	// Get the current time in the "Asia/Taipei" time zone
	currentTimeInTaipei := time.Now().In(taipeiLocation)

	// Display the local time in Taipei in customized format [HH:mm]
	return currentTimeInTaipei.Format("[15:04]"), nil
}

func getReward(bonus int) int {
	return int(math.Round(7 * (float64(bonus)/100.0 + 1)))
}

func getSortedRank(clients map[string]*Client) []*Client {
	clientAndScore := make([]*Client, 0, len(clients))
	for _, client := range clients {
		clientAndScore = append(clientAndScore, client)
	}
	sort.Slice(clientAndScore, func(i, j int) bool {
		return clientAndScore[i].score > clientAndScore[j].score
	})

	return clientAndScore
}

func (r *Room) run() {
	defer func() {
		r.mu.Lock()
		r.isClosed = true
		fmt.Println("room close")
		close(r.join)
		close(r.leave)
		close(r.BroadcastArea)
		close(r.gameControlChan)
		r.mu.Unlock()
	}()

	//outerloop:
	for {
		select {
		case client := <-r.join:
			r.mu.RLock()
			fmt.Printf("%s join\n", client.name)

			// get timeStamp
			timeStamp, err := getTimeStamp()
			if err != nil {
				fmt.Println("get time stamp error:", err)
			}
			// 如果我們要在同一個select case去觸發其他case(此例是對channel傳訊息)，就要把這個channel變成buffer channel，這樣才不會導致這則訊息塞住，而其他的訊息進不來造成訊息卡死
			r.BroadcastArea <- &Msg{Type: "chat", Payload: Data{Content: fmt.Sprintf("J%s```%s", client.name, timeStamp)}}

			// send score
			fmt.Println("Score Update")
			score := &ScoreDict{Type: "score", Dict: make(map[string]int)}
			for _, client := range r.clients {
				score.Dict[client.name] = client.score
			}
			client.receiveScoreUpdate <- score // send all existing players' score to the new commer

			// send the new commer's score to everyone
			score = &ScoreDict{Type: "score", Dict: map[string]int{client.name: 0}}
			for _, existingClient := range r.clients {
				if client.name != existingClient.name {
					existingClient.receiveScoreUpdate <- score
				}
			}

			// update the circular doubly-linked list and initial room Master
			// setting room master
			if r.numOfClients == 1 {
				r.roomMaster = &clientNode{clientName: client.name}
			} else {
				newNode := &clientNode{clientName: client.name}
				node := r.roomMaster
				for node.next != nil && node.next != r.roomMaster {
					node = node.next
				}
				node.next = newNode
				newNode.prev = node
				newNode.next = r.roomMaster
				r.roomMaster.prev = newNode
			}
			// send room master
			masterData := &Msg{Type: "roomMaster", Payload: Data{Content: r.roomMaster.clientName + "```" + fmt.Sprint(r.numOfClients)}}
			r.BroadcastArea <- masterData

			r.mu.RUnlock()

		case client := <-r.leave:
			r.mu.RLock()
			fmt.Printf("%s leave\n", client.name)
			// get timeStamp
			timeStamp, err := getTimeStamp()
			if err != nil {
				fmt.Println("get time stamp error:", err)
			}
			r.BroadcastArea <- &Msg{Type: "chat", Payload: Data{Content: fmt.Sprintf("L%s```%s", client.name, timeStamp)}}

			// send the leaving player score to everyone, set the score to -1 means he's leaving
			fmt.Println("Score Update")
			score := &ScoreDict{Type: "score", Dict: map[string]int{client.name: -1}}
			for _, existingClient := range r.clients {
				existingClient.receiveScoreUpdate <- score
			}

			// 對circular-doubly-linked list做刪除節點
			fmt.Println("maintain linked-list")
			if client.name == r.roomMaster.clientName {
				r.roomMaster = r.roomMaster.next
				prevRoomMaster := r.roomMaster.prev
				node := r.roomMaster
				for node.next != prevRoomMaster {
					node = node.next
				}
				node.next = r.roomMaster
				r.roomMaster.prev = node
			} else {
				node := r.roomMaster
				for node.next.clientName != client.name {
					node = node.next
				}
				node.next = node.next.next
				node.next.prev = node
			}
			// send msg to update roomMaster and send the numOfClients
			masterData := &Msg{Type: "roomMaster", Payload: Data{Content: r.roomMaster.clientName + "```" + fmt.Sprint(r.numOfClients)}}
			r.BroadcastArea <- masterData

			if r.isPlaying && r.numOfClients == 1 {
				// 讓遊戲直接結束，並reset
				fmt.Println("Someone leaves, Game Over")
				var c *Client
				for _, client := range r.clients {
					c = client
					break
				}
				r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d", c.name, c.score)}}
				r.isGameOver = true
				r.mu.RUnlock()
				go r.reset() //這個function出問題去影響到整個前端通訊，有connection，但沒辦法回BroadcastArea訊息
			} else {
				// 用trick的方法去處理cur_painter在作畫時離開
				// 並且如果遊戲已達勝利條件就結束
				// 否則送出RSK signal
				if !r.isRoundOver && r.isPlaying {
					if r.curPainter != nil && client.name == r.curPainter.clientName {
						if findMax(r.clients) >= r.winScore {
							clientAndScore := getSortedRank(r.clients)
							if len(clientAndScore) == 2 {
								r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d```%s```%d", clientAndScore[0].name, clientAndScore[0].score, clientAndScore[1].name, clientAndScore[1].score)}}
							} else {
								r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d```%s```%d```%s```%d", clientAndScore[0].name, clientAndScore[0].score, clientAndScore[1].name, clientAndScore[1].score, clientAndScore[2].name, clientAndScore[2].score)}}
							}
							r.isGameOver = true
							r.mu.RUnlock()
							go r.reset()
						} else {
							fmt.Println("Server sends RSK")
							go r.RSK()
							r.isRoundOver = true
							r.curPainterExit = !r.curPainterExit
							r.curPainter = r.curPainter.next
							r.mu.RUnlock()
						}
					} else {
						r.mu.RUnlock()
					}
				} else {
					r.mu.RUnlock()
				}
			}
		case msg := <-r.BroadcastArea:
			r.mu.Lock()
			fmt.Println("enter BroadcastArea")
			for _, client := range r.clients {
				// 怕在送訊息的時候對方先走了，但room還沒更新他的client map，導致送訊息給已關閉的channel
				if !client.isLeft {
					client.receive <- msg
				}
			}
			r.mu.Unlock()
			fmt.Println("leave BroadcastArea")
		/*
			case <-r.scoreUpdateChan:
				fmt.Println("Score Update")
				r.mu.Lock()
				score := &ScoreDict{Type: "score", Dict: make(map[string]int)}
				for _, client := range r.clients {
					score.Dict[client.name] = client.score
				}
				for _, client := range r.clients {
					client.receiveScoreUpdate <- score
				}
				r.mu.Unlock()
		*/
		case signal := <-r.gameControlChan:
			r.mu.RLock()
			if signal.Type == "IN" {
				if signal.Payload.Content == "0" {
					r.isInvited = true
				} else {
					r.isInvited = false
				}
			} else if string(signal.Type) == "RO" { // RO => Round Over
				// 優先計算並記錄畫家出題者這回合的得分
				r.isRoundOver = true
				painter := r.clients[r.curPainter.clientName]
				painter.score += int(float64(r.roundScore) * float64(0.2))
				r.roundScore = 0
				// check if the win condition is met
				// if someone wins, send GO => Game Over 讓遊戲結束並reset
				if findMax(r.clients) >= r.winScore {
					clientAndScore := getSortedRank(r.clients)
					if len(clientAndScore) == 2 {
						r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d```%s```%d", clientAndScore[0].name, clientAndScore[0].score, clientAndScore[1].name, clientAndScore[1].score)}}
					} else {
						r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d```%s```%d```%s```%d", clientAndScore[0].name, clientAndScore[0].score, clientAndScore[1].name, clientAndScore[1].score, clientAndScore[2].name, clientAndScore[2].score)}}
					}
					r.isGameOver = true
					go r.reset()
				} else {
					// 送出畫家得分
					score := &ScoreDict{Type: "score", Dict: map[string]int{painter.name: painter.score}}
					for _, existingClient := range r.clients {
						existingClient.receiveScoreUpdate <- score
					}
					go r.RO("0")
				}
			} else if string(signal.Type) == "RSK" { // RSK => Round Skip
				r.isRoundOver = true
				go r.RSK()
			} else if string(signal.Type) == "GS" { // GS => Game Start
				r.isPlaying = true
				r.curPainter = r.roomMaster
				// use "@" as a delimitor, so the clientName can not contain "@"
				// 記得要順便送出題目選項
				fmt.Printf("print: %s@%s\n", r.curPainter.clientName, r.curPainter.next.clientName)
				fmt.Println("Start getting questions")
				q1, q2 := getQuestions(&category, r.questionRecord)
				fmt.Println("Stop getting questions")
				r.BroadcastArea <- &Msg{Type: "GS", Payload: Data{Content: fmt.Sprintf("%s@%s@%s@%s", r.curPainter.clientName, r.curPainter.next.clientName, q1, q2)}}
			} else if string(signal.Type) == "CS" {
				answerAndIndex := strings.Split(signal.Payload.Content, "@")
				r.answer = answerAndIndex[0]
				index, _ := strconv.Atoi(answerAndIndex[1])
				// 把題目的index註冊掉，這樣有畫過題目就不會重複出現
				(*r.questionRecord)[index] = nil
				fmt.Println(r.answer)
				r.BroadcastArea <- &Msg{Type: "CS"}
			} else if string(signal.Type) == "sys" { // sys means players input at the answer area
				// check if the player guess the correct answer
				// use "```" as delimitor for message
				isAllGuessedRight := false
				playerNameAndContent := strings.Split(signal.Payload.Content, "```")
				playerName, content, timeStamp, pointReward := playerNameAndContent[0], playerNameAndContent[1], playerNameAndContent[2], playerNameAndContent[3]
				if content == r.answer {
					isAllGuessedRight = true
					client := r.clients[playerName]
					// reward logic
					// 先把倍數從字串轉為整數，再轉成float64
					// 保底分數為7分，乘上倍率最後再取整
					// 最後記錄到roundScore裡面 在回合結束時計算畫家的應得分數
					bonus, _ := strconv.Atoi(pointReward)
					reward := getReward(bonus)
					client.score += reward
					r.roundScore += reward
					// update the player's score
					score := &ScoreDict{Type: "score", Dict: map[string]int{client.name: client.score}}
					for _, existingClient := range r.clients {
						existingClient.receiveScoreUpdate <- score
					}
					// check if everyone guesses the right answer
					r.guessRight[playerName] = nil // add the player to the guessRight dict
					for clientName := range r.clients {
						if _, ok := r.guessRight[clientName]; !ok && clientName != r.curPainter.clientName {
							isAllGuessedRight = false
							break
						}
					}
					r.BroadcastArea <- &Msg{Type: "sys", Payload: Data{Content: fmt.Sprintf("G%s```%s", client.name, timeStamp)}}
				} else {
					r.BroadcastArea <- &Msg{Type: "sys", Payload: Data{Content: fmt.Sprintf("C%s```%s```%s", playerName, content, timeStamp)}}
				}

				if isAllGuessedRight {
					// 優先計算並記錄畫家出題者這回合的得分
					painter := r.clients[r.curPainter.clientName]
					painter.score += int(float64(r.roundScore) * float64(0.2))
					r.roundScore = 0
					// 判斷是否提早結束回合並剛好遊戲結束
					if findMax(r.clients) >= r.winScore {
						clientAndScore := getSortedRank(r.clients)
						if len(clientAndScore) == 2 {
							r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d```%s```%d", clientAndScore[0].name, clientAndScore[0].score, clientAndScore[1].name, clientAndScore[1].score)}}
						} else {
							r.BroadcastArea <- &Msg{Type: "GO", Payload: Data{Content: fmt.Sprintf("%s```%d```%s```%d```%s```%d", clientAndScore[0].name, clientAndScore[0].score, clientAndScore[1].name, clientAndScore[1].score, clientAndScore[2].name, clientAndScore[2].score)}}
						}
						r.isGameOver = true
						go r.reset()
					} else {
						// 送出畫家得分
						score := &ScoreDict{Type: "score", Dict: map[string]int{painter.name: painter.score}}
						for _, existingClient := range r.clients {
							existingClient.receiveScoreUpdate <- score
						}
						// 一般的提早結束回合
						r.isRoundOver = true
						go r.RO("1")
					}
				}
			}
			r.mu.RUnlock()
		case <-r.ctx.Done():
			fmt.Printf("room %s is empty\n", r.roomID)
			return
		}
	}
}

func (r *Room) reset() {
	time.Sleep(time.Second * 1) // 不確定需不需要
	r.mu.Lock()
	fmt.Println("reset")
	defer func() {
		r.mu.Unlock()
		fmt.Println("done reset")
	}()

	r.isPlaying = false
	r.isRoundOver = false
	r.isGameOver = false
	r.curPainterExit = false
	r.answer = ""
	r.guessRight = map[string]interface{}{}
	r.questionRecord = &map[int]interface{}{}
	r.otp = map[string]interface{}{}
	for len(r.join) > 0 {
		<-r.join
	}
	fmt.Println("clear join")
	for len(r.leave) > 0 {
		<-r.leave
	}
	fmt.Println("clear leave")
	for len(r.BroadcastArea) > 0 {
		<-r.BroadcastArea
	}
	fmt.Println("clear BroadcastArea")
	for len(r.gameControlChan) > 0 {
		<-r.gameControlChan
	}
	fmt.Println("clear gameControlChan")
	for _, client := range r.clients {
		client.mu.Lock()
		client.score = 0
		for len(client.receive) > 0 {
			<-client.receive
		}
		fmt.Printf("clear %s receive\n", client.name)
		for len(client.receiveScoreUpdate) > 0 {
			<-client.receiveScoreUpdate
		}
		fmt.Printf("clear %s receiveScoreUpdate\n", client.name)
		client.mu.Unlock()
	}
	/*
		// Close channels
		close(r.join)
		close(r.leave)
		close(r.BroadcastArea)
		//close(r.scoreUpdateChan)
		close(r.gameControlChan)

		// Create new channels to replace the closed ones
		r.join = make(chan *Client, 10)
		r.leave = make(chan *Client, 10)
		r.BroadcastArea = make(chan *Msg, 100)
		//r.scoreUpdateChan = make(chan struct{}, 10)
		r.gameControlChan = make(chan *Msg, 10)

		// Reset client channel and score
		for _, client := range r.clients {
			client.mu.Lock()
			client.score = 0
			close(client.receive)
			close(client.receiveScoreUpdate)
			client.receive = make(chan *Msg, 100)
			client.receiveScoreUpdate = make(chan *ScoreDict, 100)
			client.mu.Unlock()
		}
	*/
}

/*
type Room struct {
	roomID string

	numOfClients int

	clients map[*Client]bool

	join chan *Client

	leave chan *Client

	BroadcastArea chan []byte
}

func newRoom(id string) *Room {
	return &Room{
		roomID:       id,
		numOfClients: 0,
		clients:      make(map[*Client]bool),
		join:         make(chan *Client),
		leave:        make(chan *Client),
		BroadcastArea:     make(chan []byte),
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
		case msg := <-r.BroadcastArea:
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

	BroadcastArea chan []byte
}

func newRoom(id string) *Room {
	return &Room{
		roomID:       id,
		numOfClients: 0,
		clients:      make(map[*Client]bool),
		join:         make(chan *Client),
		leave:        make(chan *Client),
		BroadcastArea:     make(chan []byte),
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
		case msg := <-r.BroadcastArea:
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
	client.room.BroadcastArea <- []byte("new join")
	defer func() {
		client.room.BroadcastArea <- []byte("someone is leaving")
		r.leave <- client
	}()
	// 這裡要注意這兩行的寫法，相反會在關閉的時候出錯
	go client.WriteMsg()
	client.readMsg()

}
*/
