package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"elevator/model/common"
	"elevator/model/elevator"
	"elevator/model/user"
)

// type User struct {
// 	Name   string `json:"name"`
// 	UserId int    `json:"userId"`
// }

func TestElevatorSystem(t *testing.T) {
	ctrl := elevator.NewElevatorCtrl()
	totalRequests := 40
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	rand.Seed(time.Now().UnixNano())
	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		user := user.UserInfo{
			Name:   fmt.Sprintf("User%d", i+1),
			UserId: i + 1,
		}
		req := common.Request{
			Name: user.Name,
		}
		go func(u common.Request) {
			defer wg.Done()
			from := rand.Intn(10) + 1
			to := rand.Intn(10) + 1
			for from == to {
				to = rand.Intn(10) + 1
			}
			idleElevator := ctrl.FindIdleElevator()
			if idleElevator != nil {
				fmt.Printf("%s requests elevator from %d to %d\n", u.Name, from, to)
				// 將操作電梯的指令送往 channel
				cmd := elevator.CMD{
					Request: req,
				}
				idleElevator.MoveChan <- cmd
			}
			time.Sleep(1 * time.Second) // 每秒產生一個請求
		}(req)
	}

	wg.Wait()
	duration := time.Since(startTime).Seconds()
	t.Logf("All %d requests processed in %.2f seconds using multiple elevators", totalRequests, duration)
}
