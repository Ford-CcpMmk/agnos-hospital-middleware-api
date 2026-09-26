package router

import (
	"agnos-assignment/internal/handler"
	"agnos-assignment/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func New(
	database handler.DatabasePinger,
	staffService handler.StaffService,
	patientService handler.PatientSearcher,
	tokens middleware.TokenParser,
) *gin.Engine {
	binding.EnableDecoderDisallowUnknownFields = true

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	staffHandler := handler.NewStaffHandler(staffService)
	patientHandler := handler.NewPatientHandler(patientService)

	r.GET("/health", handler.Live)
	r.GET("/health/live", handler.Live)
	r.GET("/health/ready", handler.Ready(database))

	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.Live)
		api.POST("/staff/create", staffHandler.Create)
		api.POST("/staff/login", staffHandler.Login)

		patients := api.Group("/patient")
		patients.Use(middleware.Authenticate(tokens))
		patients.POST("/search", patientHandler.Search)
	}

	return r
}
