package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingUp
	MovingDown
	Stopped
)

type Request struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type User struct {
	Name   string `json:"name"`
	UserId int    `json:"userId"`
}

type Elevator struct {
	ID         int
	Current    int
	Direction  int
	State      ElevatorState
	Passengers []Request
	MoveChan   chan int
}

type ElevatorCtrl struct {
	elevators []Elevator
	requests  chan Request
}

func NewElevatorCtrl() *ElevatorCtrl {
	ctrl := &ElevatorCtrl{
		elevators: []Elevator{
			{ID: 1, Current: 1, State: Idle, MoveChan: make(chan int)},
			{ID: 2, Current: 1, State: Idle, MoveChan: make(chan int)},
		},
		requests: make(chan Request, 10),
	}

	for i := range ctrl.elevators {
		go ctrl.runElevator(&ctrl.elevators[i])
	}

	return ctrl
}

func (ctrl *ElevatorCtrl) FindIdleElevator() *Elevator {
	for i := range ctrl.elevators {
		if ctrl.elevators[i].State == Idle {
			return &ctrl.elevators[i]
		}
	}
	return nil
}

func (e *Elevator) move(to int) {
	if to > e.Current {
		e.State = MovingUp
	} else if to < e.Current {
		e.State = MovingDown
	} else {
		e.State = Stopped
		return
	}

	for e.Current != to {
		time.Sleep(1 * time.Second)
		if e.State == MovingUp {
			e.Current++
		} else if e.State == MovingDown {
			e.Current--
		}
		fmt.Printf("Elevator %d is at floor %d\n", e.ID, e.Current)
	}
	e.State = Stopped
	time.Sleep(1 * time.Second) // 停靠處理
}

func (ctrl *ElevatorCtrl) runElevator(e *Elevator) {
	for {
		floor := <-e.MoveChan
		e.move(floor)
	}
}

func (ctrl *ElevatorCtrl) RequestElevator(from, to int) int {
	idleElevator := ctrl.FindIdleElevator()
	if idleElevator == nil {
		fmt.Println("No idle elevator available.")
		return -1
	}
	fmt.Printf("Assigning idle elevator %d to request from %d to %d\n", idleElevator.ID, from, to)
	idleElevator.MoveChan <- from
	idleElevator.MoveChan <- to

	return idleElevator.ID
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	ctrl := NewElevatorCtrl()
	r := gin.Default()

	r.POST("/v1/elevator", func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		id := ctrl.RequestElevator(req.From, req.To)
		c.JSON(200, gin.H{"message": fmt.Sprintf("Request received id=%d", id)})
	})

	r.GET("/v1/elevator", func(c *gin.Context) {
		idleElevator := ctrl.FindIdleElevator()
		if idleElevator == nil {
			c.JSON(404, gin.H{"message": "No idle elevator available"})
			return
		}
		c.JSON(200, gin.H{"id": idleElevator.ID, "current_floor": idleElevator.Current})
	})

	r.Run(":8080")
}
