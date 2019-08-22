package main

import (
	"database/sql"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	_ "github.com/lib/pq"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
)

//const repo = "docker.pkg.github.com"
//const repo = "gcr.io"
//const repo = "registry.hub.docker.com"


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
	// TODO add response header "Www-Authenticate: Bearer realm="$HOST/token"
	// https://docs.docker.com/registry/spec/auth/token/
	r.GET("/v2/", func(c *gin.Context) {

		reader := strings.NewReader("{}")
		contentLength := reader.Size()
		extraHeaders := map[string]string{
			"Docker-Distribution-Api-Version": "registry/2.0",
			"Www-Authenticate":	`Bearer realm="https://`+ c.Request.Host + `/token",service="registry.docker.io"`,
		}

		c.DataFromReader(http.StatusUnauthorized, contentLength, "application/json; charset=utf-8", reader, extraHeaders)
	})

	// TODO change the scope, and redirect to auth.docker.io
	r.GET("/token", func(c *gin.Context) {

		//scope := c.Request.URL.Query().Get("scope")
		//
		//bp, err := lookupBuildpack(db, c.Param("namespace"), c.Param("id"))
		//if err != nil {
		//	log.Errorf("Error looking up buildpack: %q", err)
		//	c.String(http.StatusInternalServerError,
		//		fmt.Sprintf("Error looking up buildpack: %q", err))
		//	return
		//}

		//target := c.Request.URL
		//target.Scheme = "https"
		//target.Host = "auth.docker.io"
		//target.Query().Set("scope", "repository:jkutner/busybox:pull")
		//target.Query().Set("service", "registry.docker.io" )

		if target, err := url.Parse("https://auth.docker.io/token?scope=repository%3Ajkutner%2Fbusybox%3Apull&service=registry.docker.io"); err == nil {
			log.WithField("target", target.String()).Info("redirect")
			c.Redirect(http.StatusTemporaryRedirect, target.String())
			return
		}
		c.JSON(http.StatusInternalServerError, "{}")
	})

	r.GET("/v2/:namespace/:id/manifests/:tag", manifestHandler(db))
	r.GET("/v2/:namespace/:id/blobs/*extra", redirectHandler(db))
	r.HEAD("/v2/:namespace/:id/blobs/*extra", redirectHandler(db))
	r.POST("/buildpacks/", func(c *gin.Context) {
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
		if _, err := db.Exec("INSERT INTO buildpacks (namespace, id, ref, registry) VALUES ($1, $2, $3, $4)", json.Namespace, json.Id, json.Ref, json.Registry); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error inserting buildpack: %q", err))
			return
		}
		c.String(http.StatusOK, fmt.Sprintf("Created"))

	})

	r.POST("/buildpacks/:namespace/:id/manifests/:tag", createManifestHandler(db))
	_ = r.Run(":" + os.Getenv("PORT"))
}

type buildpack struct {
	Namespace string
	Id string
	Ref string
	Registry string
}

func manifestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		id := c.Param("id")
		tag := c.Param("tag")

		rows, err := db.Query("SELECT manifest FROM manifests WHERE namespace = $1 AND id = $2 AND tag = $3", namespace, id, tag)
		if err != nil {
			log.Errorf("Error looking up manifest: %q", err)
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error looking up manifest: %q", err))
			return
		}

		defer rows.Close()
		for rows.Next() {
			var manifest string
			if err := rows.Scan(&manifest); err != nil {
				log.Errorf("Error reading manifest: %q", err)
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error reading manifest: %q", err))
				return
			}
			reader := strings.NewReader(manifest)
			c.DataFromReader(http.StatusOK, reader.Size(), "application/vnd.docker.distribution.manifest.v2+json", reader, map[string]string{})
			return
		}
		c.String(http.StatusNotFound,
			fmt.Sprintf("Could not find manifest"))
		return
	}
}

//func proxyHandler(db *sql.DB) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		bp, err := lookupBuildpack(db, c.Param("namespace"), c.Param("id"))
//		if err != nil {
//			log.Errorf("Error looking up buildpack: %q", err)
//			c.String(http.StatusInternalServerError,
//				fmt.Sprintf("Error looking up buildpack: %q", err))
//			return
//		}
//
//		log.Debug(c.Request.Header)
//
//		r := c.Request
//		r.URL.Scheme = "https"
//		r.URL.Path = path.Join("/v2", bp.Ref, "blobs", c.Param("extra"))
//		r.URL.Host = bp.Registry
//		r.Host = bp.Registry
//
//		repoUrl, err := url.Parse("https://" + bp.Registry)
//		proxy := httputil.NewSingleHostReverseProxy(repoUrl)
//		proxy.ServeHTTP(c.Writer, c.Request)
//	}
//}

func redirectHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.
			WithField("headers", c.Request.Header).
			Info("headers")

		bp, err := lookupBuildpack(db, c.Param("namespace"), c.Param("id"))
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
		redirectToRegistry(c, path.Join("/v2", bp.Ref, "blobs", c.Param("extra")), bp.Registry)
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
	c.Redirect(http.StatusTemporaryRedirect, target.String())
}

func createManifestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		id := c.Param("id")
		tag := c.Param("tag")

		if _, err := db.Exec("CREATE TABLE IF NOT EXISTS manifests (namespace varchar, id varchar, tag varchar, manifest text)"); err != nil {
			log.Errorf("Error creating database table: %q", err)
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error creating database table: %q", err))
			return
		}

		_, _ = db.Exec("DELETE FROM manifests WHERE namespace = $1 AND id = $2 AND tag = $3", namespace, id, tag)

		// Read the Body content
		var bodyBytes []byte
		if c.Request.Body != nil {
			var err error
			if bodyBytes, err = ioutil.ReadAll(c.Request.Body); err != nil {
				log.Errorf("Error reading manifest: %q", err)
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error reading manifest: %q", err))
				return
			}
		}

		log.Debug(string(bodyBytes))

		log.
			WithField("namespace", namespace).
			WithField("id", id).
			WithField("tag", tag).Info("inserting")
		if _, err := db.Exec("INSERT INTO manifests (namespace, id, tag, manifest) VALUES ($1, $2, $3, $4)", namespace, id, tag, string(bodyBytes)); err != nil {
			log.Errorf("Error inserting manifest into database: %q", err)
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error inserting manifest into database: %q", err))
			return
		}
		return
	}
}