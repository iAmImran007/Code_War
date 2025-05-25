package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Room struct {
	upgrader        websocket.Upgrader
	waitingPlayers  []*Player
	pairedUsers     map[*Player]*Player
	currentProblems map[*Player]ProblemPropaty
	mu              sync.Mutex
	db              *Databse
}

type Player struct {
	conn    *websocket.Conn
	partner *Player
	send    chan []byte
	solved  bool
}

type Message struct {
	Type    string      `json:"type"`
	Status  string      `json:"status,omitempty"`
	Msg     string      `json:"msg,omitempty"`
	Problem interface{} `json:"problem,omitempty"`
	Code    string      `json:"code,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Text    string      `json:"text,omitempty"`
	From    string      `json:"from,omitempty"`
}

type SubmissionMessage struct {
	Type string `json:"type"`
	Code string `json:"code"`
}

type ChatMsg struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewRoom(db *Databse) *Room {
	return &Room{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		waitingPlayers:  []*Player{},
		pairedUsers:     make(map[*Player]*Player),
		currentProblems: make(map[*Player]ProblemPropaty),
		db:              db,
	}
}

func (rm *Room) HandleWs(w http.ResponseWriter, r *http.Request) {
	conn, err := rm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Websocket upgrader failed:", err)
		return
	}

	player := &Player{
		conn:    conn,
		partner: nil,
		send:    make(chan []byte, 10),
		solved:  false,
	}

	go rm.SendMsg(player)
	rm.AddNewPlayer(player)

	fmt.Println("New player connected")
}

func (rm *Room) SendMsg(player *Player) {
	defer func() {
		player.conn.Close()
		rm.handlePlayerDisconnect(player)
	}()

	for msg := range player.send {
		if err := player.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			fmt.Println("Write error:", err)
			break
		}
	}
}

func (rm *Room) AddNewPlayer(player *Player) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if len(rm.waitingPlayers) > 0 {
		partner := rm.waitingPlayers[0]
		rm.waitingPlayers = rm.waitingPlayers[1:]

		player.partner = partner
		partner.partner = player
		rm.pairedUsers[player] = partner
		rm.pairedUsers[partner] = player

		problem, err := GetRandomProblem(rm.db)
		if err != nil {
			fmt.Println("Error loading problem:", err)
			rm.handleErrorAndCleanup(player, partner, "Failed to load problem")
			return
		}

		rm.currentProblems[player] = *problem
		rm.currentProblems[partner] = *problem

		problemMsg := Message{
			Type:    "problem",
			Status:  "ready",
			Msg:     "Match found! Here's your problem:",
			Problem: problem,
		}

		problemJSON, err := json.Marshal(problemMsg)
		if err != nil {
			fmt.Println("Error marshalling problem:", err)
			rm.handleErrorAndCleanup(player, partner, "Internal server error")
			return
		}

		player.send <- problemJSON
		partner.send <- problemJSON

		fmt.Println("Two users paired with problem ID:", problem.ID)

		go rm.ListenForSolutions(player)
		go rm.ListenForSolutions(partner)
	} else {
		rm.waitingPlayers = append(rm.waitingPlayers, player)

		waitingMsg := Message{
			Type:   "status",
			Status: "waiting",
			Msg:    "Waiting for an opponent...",
		}

		waitingJSON, _ := json.Marshal(waitingMsg)
		player.send <- waitingJSON

		fmt.Println("Player added to waiting list")
	}
}

func (rm *Room) ListenForSolutions(player *Player) {
	defer func() {
		player.conn.Close()
		rm.handlePlayerDisconnect(player)
	}()

	for {
		_, message, err := player.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("unexpected close error from player %v\n", err)
				break
			}
			fmt.Printf("read error from player %v\n", err)
			break // Or: continue, if you want to ignore this error
		}

		var baseMsg map[string]interface{}
		if err := json.Unmarshal(message, &baseMsg); err != nil {
			fmt.Printf("jSON unmarshal error from player %v\nMessage: %s\n", err, string(message))
			continue
		}

		msgType, ok := baseMsg["type"].(string)
		if !ok {
			fmt.Printf("Invalid or missing 'type' field from playerMessage: %s\n", string(message))
			continue
		}

		switch msgType {
		case "submit":
			var submission SubmissionMessage
			if err := json.Unmarshal(message, &submission); err != nil {
				fmt.Printf("Submission unmarshal error from player %v\n", err)
				continue
			}
			rm.handleSubmission(player, submission.Code)

		case "chat":
			var chatMsg ChatMsg
			if err := json.Unmarshal(message, &chatMsg); err != nil {
				fmt.Printf("Chat unmarshal error from player %v\n", err)
				continue
			}
			rm.handleChatMsg(player, chatMsg.Text)

		default:
			fmt.Printf("Unknown message type '%s'", msgType)
		}
	}
}


func (rm *Room) handleSubmission(player *Player, code string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if player.solved {
		return
	}

	problem, exists := rm.currentProblems[player]
	if !exists {
		return
	}

	fmt.Printf("Judging submission for player with problem ID: %d\n", problem.ID)

	// Convert TestCaesPropaty to TestCase for judge function
	var testCases []TestCase
	for _, tc := range problem.TestCases {
		testCases = append(testCases, TestCase{
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
		})
	}

	result, err := JudgeCode(problem.ID, code, testCases, rm.db)
	if err != nil {
		fmt.Printf("Judge error: %v\n", err)
		errorMsg := Message{
			Type:   "error",
			Status: "error",
			Msg:    "Compilation or runtime error: " + err.Error(),
		}
		errorJSON, _ := json.Marshal(errorMsg)
		player.send <- errorJSON
		return
	}

	// Send result to the player who submitted
	resultMsg := Message{
		Type:   "result",
		Status: "judged",
		Msg:    fmt.Sprintf("Passed %d/%d test cases", result.Passed, result.Total),
		Result: result,
	}
	resultJSON, _ := json.Marshal(resultMsg)
	player.send <- resultJSON

	// Check if player won (solved all test cases)
	if result.Passed == result.Total {
		player.solved = true
		rm.handleGameWin(player)
	}
}

func (rm *Room) handleGameWin(winner *Player) {
	partner := winner.partner
	if partner == nil {
		return
	}

	// Send win message to winner
	winMsg := Message{
		Type:   "game_end",
		Status: "win",
		Msg:    "Congratulations! You won the match!",
	}
	winJSON, _ := json.Marshal(winMsg)
	winner.send <- winJSON

	// Send lose message to opponent
	loseMsg := Message{
		Type:   "game_end",
		Status: "lose",
		Msg:    "You lost! Your opponent solved the problem first.",
	}
	loseJSON, _ := json.Marshal(loseMsg)
	partner.send <- loseJSON

	fmt.Println("Game finished - winner determined")

	// Clean up after a short delay to allow messages to be sent coz 
	go func() {
		rm.CleanupPlayers(winner)
		rm.CleanupPlayers(partner)
	}()
}

func (rm *Room) handlePlayerDisconnect(player *Player) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if player.partner != nil && !player.solved {
		// If player has a partner and game is ongoing, partner wins
		partner := player.partner

		winMsg := Message{
			Type:   "game_end",
			Status: "win",
			Msg:    "You won! Your opponent disconnected.",
		}
		winJSON, _ := json.Marshal(winMsg)
		partner.send <- winJSON

		fmt.Println("Player disconnected - opponent wins by default")
	}

	rm.CleanupPlayers(player)
}

func (rm *Room) handleErrorAndCleanup(player, partner *Player, errorMsg string) {
	errMessage := Message{
		Type:   "error",
		Status: "error",
		Msg:    errorMsg,
	}

	errJSON, _ := json.Marshal(errMessage)

	player.send <- errJSON
	if partner != nil {
		partner.send <- errJSON
	}

	rm.CleanupPlayers(player)
	if partner != nil {
		rm.CleanupPlayers(partner)
	}
}

func (rm *Room) CleanupPlayers(player *Player) {
	if player.partner != nil {
		partner := player.partner

		delete(rm.pairedUsers, player)
		delete(rm.pairedUsers, partner)

		delete(rm.currentProblems, player)
		delete(rm.currentProblems, partner)

		partner.partner = nil
		player.partner = nil
	} else {
		for i, p := range rm.waitingPlayers {
			if p == player {
				rm.waitingPlayers = append(rm.waitingPlayers[:i], rm.waitingPlayers[i+1:]...)
				break
			}
		}

		delete(rm.pairedUsers, player)
		delete(rm.currentProblems, player)
	}

	select {
	case <-player.send:
	default:
		close(player.send)
	}

	fmt.Println("Cleanup complete for player")
}

func (rm *Room) handleChatMsg(player *Player, text string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
		//if both are active
		if player.partner == nil || player.solved {
			return 
		}

		//chat 
		chatMsg := Message{
			Type: "chat",
		    Text: text,
		    From: "opponent",
		}

		chatJson, err := json.Marshal(chatMsg)
		if err != nil {
			fmt.Println("Error while mershaligng chat msg")
			return
		}

		//send the msg
		select{
		case player.partner.send <- chatJson:
			fmt.Printf("Chat msg send to: %s\n", text)
		default:
			fmt.Println("Feild to send a chat msg chanel problem")
		}
	
}