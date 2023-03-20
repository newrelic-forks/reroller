package registry

import (
	"errors"
	"strings"

	"github.com/newrelic-forks/reroller/src/registry/docker"
	"github.com/newrelic-forks/reroller/src/registry/gcr"
	"github.com/newrelic-forks/reroller/src/registry/ghcr"
	"github.com/newrelic-forks/reroller/src/registry/quay"
)

func ImageDigests(image string) ([]string, error) {
	id := ParseImage(image)

	infoFunc := ImageInfoFunc(func(image, tag string) ([]string, error) {
		return nil, errors.New("unknown registry for image " + image)
	})

	switch id.Registry {
	case docker.Registry:
		infoFunc = docker.DockerLikeImageInfo(docker.BaseUrl)
	case ghcr.Registry:
		infoFunc = ghcr.GHCRLikeImageInfo(ghcr.BaseUrl)
	case quay.Registry:
		infoFunc = quay.QuayLikeImageInfo("https://quay.io/api/v1/repository")
	case gcr.Registry, "k8s.gcr.io":
		infoFunc = gcr.GCRLikeImageInfo("https://" + id.Registry + "/v2")
	}

	return infoFunc(id.Name, id.Tag)
}

// ImageInfoFunc is able to provide the latest SHA of an image given its name and tag
type ImageInfoFunc func(image, tag string) ([]string, error)

type ImageDescriptor struct {
	Registry string
	Name     string
	Tag      string
}

const defaultRegistry = "docker.io"
const defaultTag = "latest"

func ParseImage(image string) ImageDescriptor {
	d := ImageDescriptor{}

	pieces := strings.Split(image, "/")

	if len(pieces) >= 3 {
		// Has registry
		d.Registry = pieces[0]
		pieces = pieces[1:]
	} else {
		d.Registry = defaultRegistry
	}

	if len(pieces) < 2 {
		pieces[0] = "library/" + pieces[0]
	}

	repotag := strings.Split(strings.Join(pieces, "/"), ":")
	d.Name = repotag[0]
	if len(repotag) >= 2 {
		// Has tag
		d.Tag = repotag[1]
	} else {
		d.Tag = defaultTag
	}

	return d
}
