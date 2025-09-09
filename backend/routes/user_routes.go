package routes

import (
	"PoolManagerVM/backend/controllers"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	users := r.Group("/users")
	{
		users.GET("/", controllers.GetUsers)
		users.POST("/", controllers.CreateUser)
		// users.DELETE("/", controllers.DeleteUser)
	}
}
