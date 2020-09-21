package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	API = "http://localhost:8000"
	E_STATUS_STOOPED = "STOPPED"
	E_STATUS_OPEND = "OPENED"
	E_STATUS_UPWARD = "UPWARD"
	E_STATUS_DOWNWARD = "DOWNWARD"
	E_CMD_STOP = "STOP"
	E_CMD_UP = "UP"
	E_CMD_DOWN = "DOWN"
	E_CMD_OPEN = "OPEN"
	E_CMD_CLOSE = "CLOSE"
	E_CMD_ENTER = "ENTER"
	E_CMD_EXIT = "EXIT"
)

type Game struct {
	token string
	resStart *StartResponse
	eles []*Elevator
	pass map[int][]*Passenger
	isEnd bool
	actionQueue map[int]*Action
	maxFloor int
	maxCarrying int
	timestamp int

	up map[int]bool
	down map[int]bool

}

type StartResponse struct {
	Token     string     `json:"token"`
	Timestamp int        `json:"timestamp"`
	Elevators []*Elevator `json:"elevators"`
	IsEnd     bool       `json:"is_end"`
}

type OnCallsResponse struct {
	Token string `json:"token"`
	Timestamp int `json:"timestamp"`
	Elevators []*Elevator `json:"elevators"`
	Calls []*Passenger `json:"calls"`
	IsEnd bool `json:"is_end"`
}

type Elevator struct {
	Id         int         `json:"id"`
	Floor      int         `json:"floor"`
	Passengers []*Passenger `json:"passengers"`
	Status     string      `json:"status"`
}

type ActionRequest struct {
	Commands []*Command `json:"commands"`
}

type Command struct {
	ElevatorId int `json:"elevator_id"`
	Command string `json:"command"`
	CallIds []int	`json:"call_ids"`
}

type Action struct {
	Id int
	Cmd string
	CallIds []int
}

type Passenger struct {
	Id        int `json:"id"`
	Timestamp int `json:"timestamp"`
	Start     int `json:"start"`
	End       int `json:"end"`
}

func Start(problemId int, numOfEle int, maxFloor int, maxCarrying int) (*Game, error){
	res, err := http.Post(API + "/start/tester/" + strconv.Itoa(problemId) + "/" + strconv.Itoa(numOfEle), "application/json", nil)
	if err != nil {
		fmt.Println("Failed to start new game: ", err)
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("Not OK", res.StatusCode)
		return nil, err
	}
	resStart := &StartResponse{}
	if err := json.NewDecoder(res.Body).Decode(&resStart); err != nil {
		fmt.Println("Failed to parse json: ", err)
		return nil, err
	}
	newGame := &Game{
		resStart: resStart,
		token: resStart.Token,
		maxFloor: maxFloor,
		maxCarrying: maxCarrying,
		up: make(map[int]bool),
		down: make(map[int]bool),
		timestamp: resStart.Timestamp,
	}
	if numOfEle == 1 {
		newGame.up[0] = true
	} else if numOfEle == 2 {
		newGame.up[0] = true
		newGame.down[1] = true
	} else if numOfEle == 3 {
		newGame.up[0] = true
		newGame.down[1] = true
		newGame.up[2] = true
	} else if numOfEle == 4 {
		newGame.up[0] = true
		newGame.down[1] = true
		newGame.up[2] = true
		newGame.down[3] = true
	}

	return newGame, nil
}

func (g *Game) Status() {
	fmt.Println("==========")
	fmt.Println("\t timestamp: ", g.timestamp)
	fmt.Println("\t token: ", g.token)
	fmt.Println("\t isEnd: ", g.isEnd)
	fmt.Println("\t Elevator status: ")
	for _, e := range g.eles {
		fmt.Printf("\t\t id: %d status: %s floor: %d carrying: %d\n", e.Id, e.Status, e.Floor, len(e.Passengers))
		for _, p := range e.Passengers {
			fmt.Printf("\t\t\t id: %d start: %d end: %d timestamp: %d\n", p.Id, p.Start, p.End, p.Timestamp)
		}
	}
	fmt.Println("\t Calls status: ")
	for _, pList := range g.pass {
		for _, p := range pList {
			fmt.Printf("\t\t\t id: %d start: %d end: %d timestamp: %d\n", p.Id, p.Start, p.End, p.Timestamp)
		}
	}

	fmt.Println("==========")

}

func (g *Game) OnCalls() {
	client := &http.Client{}
	if req, err := http.NewRequest(http.MethodGet, API + "/oncalls", nil); err != nil {
		fmt.Println(err)
		return
	} else {
		req.Header.Add("X-Auth-Token", g.token)
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			panic("Failed to oncalls " +  err.Error())
		}
		resOnCalls := &OnCallsResponse{}
		err = json.NewDecoder(res.Body).Decode(&resOnCalls)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(resOnCalls)
		g.eles = resOnCalls.Elevators
		g.pass = make(map[int][]*Passenger)
		for _, p := range resOnCalls.Calls {
			if _, exist := g.pass[p.Start]; !exist {
				g.pass[p.Start] = make([]*Passenger, 0)
			}
			g.pass[p.Start] = append(g.pass[p.Start], p)
		}
		g.isEnd = resOnCalls.IsEnd
		g.timestamp = resOnCalls.Timestamp
	}
}

func (g *Game) ClearActionQueue() {
	g.actionQueue = make(map[int]*Action)
}

func (g *Game) PutAction(eleId int, cmd string, calls []int) {
	g.actionQueue[eleId] = &Action{
		Id: eleId,
		Cmd: cmd,
		CallIds: calls,
	}
}

func (g *Game) DoAction() {
	requestBody := make([]*Command, 0, len(g.actionQueue))
	for _, a := range g.actionQueue {
		cmd := &Command{
			ElevatorId: a.Id,
			Command: a.Cmd,
			CallIds: nil,
		}
		if a.Cmd == E_CMD_ENTER || a.Cmd == E_CMD_EXIT {
			cmd.CallIds = a.CallIds
		}
		requestBody = append(requestBody, cmd)
	}
	for _, v := range requestBody {
		fmt.Println("CMD: ", v)
	}
	apiAction := &ActionRequest{
		Commands: requestBody,
	}
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(&apiAction); err != nil {
		fmt.Println(err)
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, API + "/action", body);
	if err != nil {
		fmt.Println("failed to making new request", err)
		return
	}
	req.Header.Add("X-Auth-Token", g.token)

	if res, err := client.Do(req); err != nil {
		fmt.Println("failed to post: ", err)
	} else  {
		fmt.Println("res: ", res, body)
		if res.StatusCode != http.StatusOK {
			fmt.Println("not ok: ", res.StatusCode)
			return
		}
		resStart := &StartResponse{}
		if err := json.NewDecoder(res.Body).Decode(&resStart); err != nil {
			fmt.Println("failed to decode: ", err)
		} else {
			g.isEnd = resStart.IsEnd
		}
	}
}


func (g *Game) isExistExitPassengers(e *Elevator) bool {
	if len(e.Passengers) == 0  {
		return false
	}
	for _, p := range e.Passengers {
		if p.End == e.Floor {
			return true
		}
	}
	return false
}

func (g *Game) isExistEnterPassengers(e *Elevator) bool {
	if len(g.pass[e.Floor]) == 0 || len(e.Passengers) >= g.maxCarrying {
		return false
	}
	for _, p := range g.pass[e.Floor] {
		if _, exist := g.up[e.Id]; exist {
			if isUpGoing(p) {
				return true
			}
		} else {
			if isDownGoing(p) {
				return true
			}
		}
	}
	return false
}

func isUpGoing(p *Passenger)  bool {
	return p.Start < p.End
}

func isDownGoing(p *Passenger) bool {
	return p.Start > p.End
}

func (g *Game) enterPassengers(e *Elevator) []int {
	calls := make([]int, 0)
	left := make([]*Passenger, 0, len(g.pass[e.Floor]))
	for _, p := range g.pass[e.Floor] {
		if (g.up[e.Id] && isUpGoing(p)) || (g.down[e.Id] && isDownGoing(p)) {
			if len(e.Passengers) + len(calls) >= g.maxCarrying {
				left = append(left, p)
			} else {
				calls = append(calls, p.Id)
			}
		} else {
			left = append(left, p)
		}
	}
	g.pass[e.Floor] = left
	return calls
}

func (g *Game) exitPassengers(e *Elevator) []int {
	calls := make([]int, 0)
	for _, p := range e.Passengers {
		if p.End == e.Floor {
			calls = append(calls, p.Id)
		}
	}
	return calls
}

func (g *Game) stopElevator(e *Elevator) bool {
	if e.Status == E_STATUS_STOOPED || e.Status == E_STATUS_OPEND {
		return true
	}
	g.PutAction(e.Id, E_CMD_STOP, nil)
	return false
}

func (g *Game) openElevator(e *Elevator) bool {
	if e.Status == E_STATUS_OPEND {
		return true
	}
	g.PutAction(e.Id, E_CMD_OPEN, nil)
	return false
}

func (g *Game) enterElevator(e *Elevator, calls []int) bool {
	g.PutAction(e.Id, E_CMD_ENTER, calls)
	return false
}

func (g *Game) exitElevator(e *Elevator, calls []int) bool {
	g.PutAction(e.Id, E_CMD_EXIT, calls)
	return false
}

func (g *Game) closeElevator(e *Elevator) bool {
 	if e.Status == E_STATUS_STOOPED  || e.Status ==E_STATUS_UPWARD || e.Status == E_STATUS_DOWNWARD {
		return true
	}
	g.PutAction(e.Id, E_CMD_CLOSE, nil)
 	return false
}

func (g *Game) moveElevator(e *Elevator) {
	if _, exist := g.up[e.Id]; exist {
		if e.Floor == g.maxFloor {
			if g.stopElevator(e) {
				g.down[e.Id] = true
				delete(g.up, e.Id)
			}
		} else {
			g.PutAction(e.Id, E_CMD_UP, nil)
		}
		return
	}

	if _, exist := g.down[e.Id]; exist {
		if e.Floor == 1 {
			if g.stopElevator(e) {
				g.up[e.Id] = true
				delete(g.down, e.Id)
			}
		} else {
			g.PutAction(e.Id, E_CMD_DOWN, nil)
		}
		return
	}
}

func Swipping(game *Game) {
	for _, e := range game.eles {
		if game.isExistExitPassengers(e) || game.isExistEnterPassengers(e) {
			if game.stopElevator(e) {
				if game.openElevator(e) {
					if game.isExistExitPassengers(e) {
						game.exitElevator(e, game.exitPassengers(e))
					} else if game.isExistEnterPassengers(e) {
						game.enterElevator(e, game.enterPassengers(e))
					} else {
						if game.closeElevator(e) {
							game.moveElevator(e)
						}
					}
				}
			}
		} else {
			if game.closeElevator(e) {
				game.moveElevator(e)
			}
		}
	}
}


func SolveProblem1(numberOfEle int) {
	game, err := Start(1, numberOfEle, 25, 8)
	if err != nil {
		panic(fmt.Sprintf("Can't start new game %v", err))
	}
	for !game.isEnd {
		game.OnCalls()
		game.ClearActionQueue()
		//game.Status()
		Swipping(game)
		game.DoAction()
	}

	game.Status()
}

func SolveProblem2(numOfEle int) {
	game, err := Start(2, numOfEle, 25, 8)
	if err != nil {
		panic(fmt.Sprintf("Can't start new game %v", err))
	}
	for !game.isEnd {
		game.OnCalls()
		game.ClearActionQueue()
		//game.Status()
		Swipping(game)
		game.DoAction()
	}

	game.Status()
}


func main() {
	//SolveProblem1(4)
	SolveProblem2(4)
}
