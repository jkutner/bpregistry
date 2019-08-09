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
	r.GET("/v2/:namespace/:repository/*extra", redirectHandler(db))
	r.HEAD("/v2/:namespace/:repository/*extra", redirectHandler(db))
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

		_, _ = db.Exec("DELETE FROM buildpacks WHERE namespace = $1 AND id = $2", json.Namespace, json.Id)

		log.
			WithField("namespace", json.Namespace).
			WithField("id", json.Id).
			WithField("ref", json.Ref).Info("creating")
		if _, err := db.Exec("INSERT INTO buildpacks (namespace, id, ref, registry) VALUES ($1, $2, $3, $4)", json.Namespace, json.Id, json.Ref, "registry.hub.docker.com"); err != nil {
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

func redirectHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bp, err := lookupBuildpack(db, c.Param("namespace"), c.Param("repository"))
		if err != nil {
			log.Errorf("Error looking up buildpack: %q", err)
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error looking up buildpack: %q", err))
			return
		}

		log.
			WithField("namespace", bp.Namespace).
			WithField("id", bp.Id).
			WithField("ref", bp.Ref).
			WithField("registry", bp.Registry).
			Info("handler")
		redirectToRegistry(c, path.Join("/v2", bp.Ref, c.Param("extra")), bp.Registry)
	}
}

func lookupBuildpack(db *sql.DB, namespace, id string) (buildpack, error) {
	rows, err := db.Query("SELECT namespace, id, ref, registry FROM buildpacks WHERE namespace = $1 AND id = $2", namespace, id)
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

	log.WithField("target", target.String()).Info("redirect")
	c.Redirect(http.StatusMovedPermanently, target.String())
}