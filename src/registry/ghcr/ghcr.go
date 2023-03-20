package ghcr

import (
	"github.com/newrelic-forks/reroller/src/registry/docker"
)

const (
	Registry = "ghcr.io"
	BaseUrl  = "https://ghcr.io/v2"
)

func GHCRLikeImageInfo(baseUrl string) func(image, tag string) ([]string, error) {
	// GHCR behaves just like docker
	return docker.DockerLikeImageInfo(baseUrl)
}
