package main

import (
	"database/sql"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"

	_ "github.com/lib/pq"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
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
		redirectToRegistry(c, "/v2/", "registry.hub.docker.com") // TODO: how redirect to the right registry with no buildpack info?
	})
	r.GET("/v2/:namespace/:repository/*extra", func(c *gin.Context) {
		bp, err := lookupBuildpack(db, c.Param("namespace"), c.Param("repository"))
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error looking up buildpack: %q", err))
			return
		}

		redirectToRegistry(c, path.Join("/v2", bp.Ref, c.Param("extra")), bp.Registry)
	})
	r.HEAD("/v2/:namespace/:repository/*extra", func(c *gin.Context) {
		bp, err := lookupBuildpack(db, c.Param("namespace"), c.Param("repository"))
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error looking up buildpack: %q", err))
			return
		}

		redirectToRegistry(c, path.Join("/v2", bp.Ref, c.Param("extra")), bp.Registry)
	})
	r.POST("/buildpacks/*extra", func(c *gin.Context) {
		var json buildpack
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, err := db.Exec("CREATE TABLE IF NOT EXISTS buildpacks (namespace varchar, id varchar, ref varchar, registry varchar)"); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error creating database table: %q", err))
			return
		}

		log.
			WithField("namespace", json.Namespace).
			WithField("id", json.Id).
			WithField("ref", json.Ref).Info("creating")
		if _, err := db.Exec("INSERT INTO buildpacks (namespace, id, ref, registry) VALUES (?, ?, ?, ?)", json.Namespace, json.Id, json.Ref, "registry.hub.docker.com"); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error inserting buildpack: %q", err))
			return
		}
		c.String(http.StatusOK, fmt.Sprintf("Created"))
	})
	_ = r.Run(":" + os.Getenv("PORT"))
}

type buildpack struct {
	Namespace string
	Id string
	Ref string
	Registry string
}

func lookupBuildpack(db *sql.DB, namespace, id string) (buildpack, error) {
	rows, err := db.Query("SELECT namespace, id, ref, registry FROM buildpacks WHERE namespace = ? AND id = ?", namespace, id)
	if err != nil {
		return buildpack{}, err
	}

	defer rows.Close()
	for rows.Next() {
		bp := buildpack{}
		if err := rows.Scan(&bp.Namespace, &bp.Id, &bp.Ref, &bp.Registry); err != nil {
			return buildpack{}, err
		}
		return bp, nil
	}
	return buildpack{}, errors.New("could not find buildpack")
}

func redirectToRegistry(c *gin.Context, repoPath, registry string) {
	target := c.Request.URL
	target.Scheme = "https"
	target.Path = repoPath
	target.Host = registry

	log.Info(target.String())
	c.Redirect(http.StatusMovedPermanently, target.String())
}