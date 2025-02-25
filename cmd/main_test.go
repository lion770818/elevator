package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestElevatorSystem(t *testing.T) {
	ctrl := NewElevatorCtrl()
	totalRequests := 40
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	rand.Seed(time.Now().UnixNano())
	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		user := User{
			Name:   fmt.Sprintf("User%d", i+1),
			UserId: i + 1,
		}
		req := Request{
			Name: user.Name,
		}
		go func(u User) {
			defer wg.Done()
			// 隨機決定 樓層跟目的
			req.From = rand.Intn(10) + 1
			req.To = rand.Intn(10) + 1
			for req.From == req.To {
				req.To = rand.Intn(10) + 1
			}
			// 尋找空閑的電梯
			idleElevator := ctrl.FindIdleElevator()
			if idleElevator != nil {
				fmt.Printf("%s (UserID: %d) requests elevator from %d to %d\n", u.Name, u.UserId, req.From, req.To)
				// 將操作電梯的指令送往 channel
				cmd := CMD{
					Request: req,
				}
				idleElevator.MoveChan <- cmd
			}
			time.Sleep(1 * time.Second) // 每秒產生一個請求
		}(user)
	}

	wg.Wait()
	duration := time.Since(startTime).Seconds()
	t.Logf("All %d requests processed in %.2f seconds using multiple elevators", totalRequests, duration)
}
