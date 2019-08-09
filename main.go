package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
)


func init() {
	// Setting default level to debug
	log.SetLevel(log.DebugLevel)
}

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	r := gin.Default()
	r.GET("/v2/", func(c *gin.Context) {
		redirectToDockerhub(c, "/v2/")
	})
	r.GET("/v2/:namespace/:repo/*extra", func(c *gin.Context) {
		// TODO maybe rewrite namespace/repo?
		namespace := c.Param("namespace")
		repository := c.Param("repository")
		extra := c.Param("extra")
		p := path.Join("/v2", namespace, repository, extra)
		redirectToDockerhub(c, p)
	})
	r.HEAD("/v2/:namespace/:repository/*extra", func(c *gin.Context) {
		// TODO maybe rewrite namespace/repo?
		namespace := c.Param("namespace")
		repository := c.Param("repository")
		extra := c.Param("extra")
		p := path.Join("/v2", namespace, repository, extra)
		redirectToDockerhub(c, p)
	})
	r.POST("/buildpacks/*extra", func(c *gin.Context) {
		if _, err := db.Exec("CREATE TABLE IF NOT EXISTS buildpacks (id varchar, ref varchar, registry varchar)"); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error creating database table: %q", err))
			return
		}

		if _, err := db.Exec("INSERT INTO buildpacks VALUES (now())"); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error incrementing tick: %q", err))
			return
		}
		c.String(http.StatusOK, fmt.Sprintf("Created"))
	})
	_ = r.Run(":" + os.Getenv("PORT"))
}

func redirectToDockerhub(c *gin.Context, repoPath string) {
	target := c.Request.URL
	target.Scheme = "https"
	target.Path = repoPath
	target.Host = "registry.hub.docker.com"
	c.Redirect(http.StatusMovedPermanently, target.String())
}