package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ElevatorState int

var (
	ElevatorMax = 2  // 電梯數量
	RequestMax  = 10 // 最大請求數量
)

// 電梯狀態機
const (
	Idle ElevatorState = iota
	MovingUp
	MovingDown
	Stopped
)

// 封包請求
type Request struct {
	Name string `json:"name"` // 搭乘的用戶
	From int    `json:"from"` // 從哪一層開始搭
	To   int    `json:"to"`   // 目的樓層
}

// 用戶
type User struct {
	Name   string `json:"name"`
	UserId int    `json:"userId"`
}

// 電梯指令
type CMD struct {
	Request
}

// 電梯
type Elevator struct {
	ID        int           // 電梯編號
	Current   int           // 目前樓層
	Direction int           // 目的
	State     ElevatorState // 狀態

	MoveChan chan CMD      // 控制電梯的 channel buffer
	Lock     *sync.RWMutex // 電梯資源鎖
}

// 電梯控制
type ElevatorCtrl struct {
	elevators []Elevator
	requests  chan Request
}

// 產生電梯控制器
func NewElevatorCtrl() *ElevatorCtrl {

	// 初始化電梯
	var elevators []Elevator
	for i := 0; i < ElevatorMax; i++ {
		elevator := Elevator{
			ID:       i,
			Current:  1,
			State:    Idle,
			MoveChan: make(chan CMD),
			Lock:     &sync.RWMutex{},
		}
		elevators = append(elevators, elevator)
	}

	// 初始化電梯控制器
	ctrl := &ElevatorCtrl{
		elevators: elevators,
		requests:  make(chan Request, RequestMax),
	}

	// 每部電梯獨立的 gorontine, 來控制移動
	for i := range ctrl.elevators {
		go ctrl.runElevator(&ctrl.elevators[i])
	}

	return ctrl
}

// 尋找閒置的電梯
func (ctrl *ElevatorCtrl) FindIdleElevator() *Elevator {

	for i := range ctrl.elevators {
		state := ctrl.elevators[i].getState()
		if state == Idle || state == Stopped {
			return &ctrl.elevators[i]
		}
	}
	return nil
}

func (e *Elevator) move(cmd CMD) {

	// 判斷電梯移動方向
	if cmd.To > e.Current {
		// The user wants to take the elevator up
		fmt.Printf("the user wants to take the elevator up... , id:%d, name:%s is at currentFloor %d\n", e.ID, cmd.Name, e.Current)
		e.setState(MovingUp)
	} else if cmd.To < e.Current {
		fmt.Printf("the user wants to take the elevator down... , id:%d, name:%s is at currentFloor %d\n", e.ID, cmd.Name, e.Current)
		e.setState(MovingDown)
	} else {
		fmt.Printf("The current status is stopped..., id:%d, name:%s is at currentFloor %d\n", e.ID, cmd.Name, e.Current)
		e.setState(Stopped)
		return
	}

	// for迴圈讓電梯移動
	for e.Current != cmd.To {
		time.Sleep(1 * time.Second)
		if e.State == MovingUp {
			// 電梯往上
			e.up()
		} else if e.State == MovingDown {
			// 電梯往下
			e.down()
		}
		fmt.Printf("elevator running id:%d, name:%s is at floor %d\n", e.ID, cmd.Name, e.Current)
	}

	// 電梯停止
	fmt.Printf("elevator stop id:%d, name:%s is at floor %d\n", e.ID, cmd.Name, e.Current)
	e.reset()
	time.Sleep(1 * time.Second) // 停靠處理
}

func (e *Elevator) setState(state ElevatorState) {

	// 設定狀態時, 不允許其他goroutine競爭
	e.Lock.Lock()
	defer e.Lock.Unlock()

	// 設定狀態
	e.State = state
}

func (e *Elevator) getState() (state ElevatorState) {

	// 讀取狀態時, 使用 read 鎖, 可以提高效能
	e.Lock.RLock()
	defer e.Lock.RUnlock()

	// 返回狀態
	return e.State
}

func (e *Elevator) reset() {

	// 設定狀態時, 不允許其他goroutine競爭
	e.Lock.Lock()
	defer e.Lock.Unlock()

	// 設定預設狀態
	e.State = Stopped
	e.Current = 1
}

func (e *Elevator) down() {

	// 設定狀態時, 不允許其他goroutine競爭
	e.Lock.Lock()
	defer e.Lock.Unlock()

	// 電梯往下
	e.Current--
}

func (e *Elevator) up() {

	// 設定狀態時, 不允許其他goroutine競爭
	e.Lock.Lock()
	defer e.Lock.Unlock()

	// 電梯往上
	e.Current++
}

func (ctrl *ElevatorCtrl) runElevator(e *Elevator) {
	for {
		// 接收指令 移動電梯
		cmd := <-e.MoveChan
		e.move(cmd)
	}
}

func (ctrl *ElevatorCtrl) RequestElevator(req Request) int {
	// 尋找閒置的電梯
	idleElevator := ctrl.FindIdleElevator()
	if idleElevator == nil {
		fmt.Println("no idle elevator available.")
		return -1
	}
	fmt.Printf("assigning idle elevator %d to request, user:%s, from %d to %d\n", idleElevator.ID, req.Name, req.From, req.To)

	// 將操作電梯的指令送往 channel
	cmd := CMD{
		Request: req,
	}
	idleElevator.MoveChan <- cmd

	return idleElevator.ID
}

func main() {

	ctrl := NewElevatorCtrl()

	r := gin.Default()
	api := r.Group("/v1")
	api.POST("/elevator", func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id := ctrl.RequestElevator(req)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("request received id=%d", id)})
	})

	api.GET("/elevator", func(c *gin.Context) {
		idleElevator := ctrl.FindIdleElevator()
		if idleElevator == nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "No idle elevator available"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": idleElevator.ID, "current_floor": idleElevator.Current})
	})

	r.Run(":8081")
}
