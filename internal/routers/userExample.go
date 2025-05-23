package routers

import (
	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/internal/handler"
)

func init() {
	apiV1RouterFns = append(apiV1RouterFns, func(group *gin.RouterGroup) {
		userExampleRouter(group, handler.NewUserExampleHandler())
	})
}

func userExampleRouter(group *gin.RouterGroup, h handler.UserExampleHandler) {
	g := group.Group("/userExample")

	// All the following routes use jwt authentication, you also can use middleware.Auth(middleware.WithExtraVerify(fn))
	//g.Use(middleware.Auth())

	// If jwt authentication is not required for all routes, authentication middleware can be added
	// separately for only certain routes. In this case, g.Use(middleware.Auth()) above should not be used.

	g.POST("/", h.Create)          // [post] /api/v1/userExample
	g.DELETE("/:id", h.DeleteByID) // [delete] /api/v1/userExample/:id
	g.PUT("/:id", h.UpdateByID)    // [put] /api/v1/userExample/:id
	g.GET("/:id", h.GetByID)       // [get] /api/v1/userExample/:id
	g.POST("/list", h.List)        // [post] /api/v1/userExample/list
}
