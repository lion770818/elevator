package main

import (
	"fmt"
	"net/http"

	"elevator/model/common"
	"elevator/model/elevator"

	"github.com/gin-gonic/gin"
)

func main() {

	// 電梯初始化
	ctrl := elevator.NewElevatorCtrl()

	r := gin.Default()
	api := r.Group("/v1")
	api.POST("/elevator", func(c *gin.Context) {
		var req common.Request
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

	r.Run(":8080")
}
