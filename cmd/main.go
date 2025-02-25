package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

type User struct {
	Name   string `json:"name"`
	UserId int    `json:"userId"`
}

// 封包請求
type Request struct {
	Name string `json:"name"` // 搭乘的用戶
	From int    `json:"from"` // 從哪一層開始搭
	To   int    `json:"to"`   // 目的樓層
}

// 電梯指令
type CMD struct {
	Request
}

// 電梯
type Elevator struct {
	ID       int           // 電梯編號
	Current  int           // 目前樓層
	State    ElevatorState // 狀態
	MoveChan chan CMD      // 控制電梯的 channel buffer
	Lock     *sync.RWMutex // 電梯資源鎖
}

// 電梯控制
type ElevatorCtrl struct {
	elevators []Elevator
	requests  chan Request
	wg        sync.WaitGroup // 用於追蹤電梯狀態
	stopChan  chan struct{}  // 用於通知電梯關閉
}

// 產生電梯控制器
func NewElevatorCtrl() *ElevatorCtrl {
	ctrl := &ElevatorCtrl{
		elevators: make([]Elevator, ElevatorMax),
		requests:  make(chan Request, RequestMax),
		stopChan:  make(chan struct{}),
	}

	// 初始化電梯
	for i := 0; i < ElevatorMax; i++ {
		ctrl.elevators[i] = Elevator{
			ID:       i,
			Current:  1,
			State:    Idle,
			MoveChan: make(chan CMD),
			Lock:     &sync.RWMutex{},
		}

		// 啟動電梯 goroutine
		ctrl.wg.Add(1)
		go ctrl.runElevator(&ctrl.elevators[i])
	}

	return ctrl
}

// runElevator 監聽 MoveChan，並處理請求
func (ctrl *ElevatorCtrl) runElevator(e *Elevator) {
	defer ctrl.wg.Done()

	for {
		select {
		case cmd := <-e.MoveChan:
			e.move(cmd)
		case <-ctrl.stopChan:
			log.Printf("電梯 %d 接收到關閉信號，等待當前請求完成...\n", e.ID)
			// 清空 MoveChan
			for len(e.MoveChan) > 0 {
				cmd := <-e.MoveChan
				e.move(cmd)
			}
			log.Printf("電梯 %d 已關閉\n", e.ID)
			return
		}
	}
}

func (e *Elevator) getState() ElevatorState {

	e.Lock.RLock()
	defer e.Lock.RUnlock()

	return e.State
}

func (e *Elevator) setState(state ElevatorState) {

	e.Lock.Lock()
	defer e.Lock.Unlock()

	e.State = state
}

func (e *Elevator) up() {

	e.Lock.Lock()
	defer e.Lock.Unlock()

	e.Current++
}

func (e *Elevator) down() {

	e.Lock.Lock()
	defer e.Lock.Unlock()

	e.Current--
}

func (e *Elevator) reset() {

	e.Lock.Lock()
	defer e.Lock.Unlock()

	e.Current = 1
	e.State = Stopped
}

func (ctrl *ElevatorCtrl) RequestElevator(req Request) int {
	idleElevator := ctrl.FindIdleElevator()
	if idleElevator == nil {
		fmt.Println("no idle elevator available.")
		return -1
	}
	log.Printf("assigning idle elevator %d to request, user:%s, from %d to %d\n", idleElevator.ID, req.Name, req.From, req.To)

	cmd := CMD{Request: req}
	idleElevator.MoveChan <- cmd
	return idleElevator.ID
}

// FindIdleElevator 找到閒置的電梯
func (ctrl *ElevatorCtrl) FindIdleElevator() *Elevator {
	for i := range ctrl.elevators {
		if ctrl.elevators[i].getState() == Idle || ctrl.elevators[i].getState() == Stopped {
			return &ctrl.elevators[i]
		}
	}
	return nil
}

// move 控制電梯移動
func (e *Elevator) move(cmd CMD) {
	if cmd.To > e.Current {
		log.Printf("電梯 %d 上升中，使用者: %s，當前樓層: %d\n", e.ID, cmd.Name, e.Current)
		e.setState(MovingUp)
	} else if cmd.To < e.Current {
		log.Printf("電梯 %d 下降中，使用者: %s，當前樓層: %d\n", e.ID, cmd.Name, e.Current)
		e.setState(MovingDown)
	} else {
		log.Printf("電梯 %d 停止，使用者: %s\n", e.ID, cmd.Name)
		e.setState(Stopped)
		return
	}

	for e.Current != cmd.To {
		time.Sleep(1 * time.Second)
		if e.State == MovingUp {
			e.up()
		} else if e.State == MovingDown {
			e.down()
		}
		log.Printf("電梯 %d 運行中，使用者: %s，當前樓層: %d\n", e.ID, cmd.Name, e.Current)
	}

	log.Printf("電梯 %d 停止，使用者: %s\n", e.ID, cmd.Name)
	e.reset()
	time.Sleep(1 * time.Second)
}

// 優雅關閉服務
func (ctrl *ElevatorCtrl) Shutdown() {
	log.Println("正在關閉電梯系統... start")
	ctrl.stopChan <- struct{}{}
	log.Println("Wait....")
	ctrl.wg.Wait() // 等待所有電梯結束
	log.Println("close channel....")
	close(ctrl.stopChan) // 通知所有電梯關閉
	log.Println("所有電梯已關閉，服務停止。")
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

	// 創建 HTTP Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// 監聽關閉信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// 接收 ctrl + c 或 kill 訊號
		<-quit
		log.Println("\n接收到關閉信號，準備關閉服務...")

		// 優雅關閉 HTTP Server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 優雅關閉電梯系統
		ctrl.Shutdown()
		log.Println("\n接收到關閉信號，關閉服務完成...")

		log.Println("關閉 http server 111 ")
		srv.Shutdown(ctx)
		log.Println("關閉 http server 222")

	}()

	log.Println("電梯服務啟動中...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("服務啟動失敗: %v\n", err)
	}
}
