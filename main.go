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
	r.GET("/v2/*repo", func(c *gin.Context) {
		target := c.Request.URL
		target.Scheme = "https"
		target.Host = "registry.hub.docker.com"
		log.Info(target.String())
		c.Redirect(http.StatusMovedPermanently, target.String())
	})
	r.HEAD("/v2/*repo", func(c *gin.Context) {
		target := c.Request.URL
		target.Scheme = "https"
		target.Host = "registry.hub.docker.com"
		log.Info(target.String())
		c.Redirect(http.StatusMovedPermanently, target.String())
	})
	_ = r.Run(":" + os.Getenv("PORT"))
}