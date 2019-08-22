# Buildpack Registry Prototype

This repo is used to prototype and test functions of a buildpack registry.

## Usage

```sh-session
$ heroku create
$ heroku addons:create heroku-postgresql
$ git push heroku master
```

Create a buildpack:

Push to Docker Hub (the only backend supported right now), and then publish the manifest to the Buildpack Registry:

```sh-session
$ docker push johndoe/some-cnb
$ curl -d '{"namespace":"me","id":"node","ref":"johndoe/some-cnb", "registry":"registry.docker.io"}' myapp.herokuapp.com/buildpacks/
$ curl -d "$(docker manifest inspect example.com/johndoe/some-cnb)" myapp.herokuapp.com/buildpacks/me/node/manifests/latest
```

Pull the buildpack:

```sh-session
$ docker pull myapp.herokuapp.com/me/node
```

## Constraints

* No buildpack versions
* Only works with Docker Hub for now
* Only works with `:latest` tag of image