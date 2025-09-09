package routes

import (
	"PoolManagerVM/backend/controllers"

	"github.com/gin-gonic/gin"
)

func ServerpoolRoutes(r *gin.Engine) {
	serverpool := r.Group("/serverpool")
	{
		serverpool.GET("/", controllers.GetServerpool)
		// serverpool.POST("/", controllers.CreateServerpool)
	}
}
