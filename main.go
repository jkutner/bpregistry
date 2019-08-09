package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"os"
)


func init() {
	// Setting default level to debug
	log.SetLevel(log.DebugLevel)
}

func main() {
	r := gin.Default()
	r.GET("/v2/", func(c *gin.Context) {
		redirectToDockerhub(c)
	})
	r.GET("/v2/:namespace/*repo", func(c *gin.Context) {
		redirectToDockerhub(c)
	})
	r.HEAD("/v2/:namespace/*repo", func(c *gin.Context) {
		redirectToDockerhub(c)
	})
	_ = r.Run(":" + os.Getenv("PORT"))
}

func redirectToDockerhub(c *gin.Context) {
	target := c.Request.URL
	target.Scheme = "https"
	target.Host = "registry.hub.docker.com"
	c.Redirect(http.StatusMovedPermanently, target.String())
}