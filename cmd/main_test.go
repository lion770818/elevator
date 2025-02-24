package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// type User struct {
// 	Name   string `json:"name"`
// 	UserId int    `json:"userId"`
// }

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
		go func(u User) {
			defer wg.Done()
			from := rand.Intn(10) + 1
			to := rand.Intn(10) + 1
			for from == to {
				to = rand.Intn(10) + 1
			}
			idleElevator := ctrl.FindIdleElevator()
			if idleElevator != nil {
				fmt.Printf("%s (UserID: %d) requests elevator from %d to %d\n", u.Name, u.UserId, from, to)
				idleElevator.MoveChan <- from
				idleElevator.MoveChan <- to
			}
			time.Sleep(1 * time.Second) // 每秒產生一個請求
		}(user)
	}

	wg.Wait()
	duration := time.Since(startTime).Seconds()
	t.Logf("All %d requests processed in %.2f seconds using multiple elevators", totalRequests, duration)
}
