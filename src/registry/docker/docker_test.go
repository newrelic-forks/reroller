package docker_test

import (
	"testing"

	"github.com/newrelic-forks/reroller/src/registry/docker"
	"github.com/stretchr/testify/require"
)

func Test_Dockerhub(t *testing.T) {
	infofunc := docker.DockerLikeImageInfo(docker.BaseUrl)

	digests, err := infofunc("newrelic/infrastructure-bundle", "1.0.0")
	require.NoError(t, err)
	require.Contains(t, digests, "sha256:7300aec653dbe8aaef96b92d2f8319a2cef8e03e99b4420ba846b46a1447ea6b")
}

func Test_Dockerhub_oci_image(t *testing.T) {
	infofunc := docker.DockerLikeImageInfo(docker.BaseUrl)

	// test buildkit image https://github.com/moby/buildkit/issues/2251
	digests, err := infofunc("newrelic/infrastructure-bundle", "3.1.3")
	require.NoError(t, err)
	require.Contains(t, digests, "sha256:d52446ef1513dea9f5928937a268afbf8583e2877b53242345505e8b7eaca78d")
	require.Contains(t, digests, "sha256:500ea42793487b8251d01047d843f971590da91dd6ca257c913e04adbd9e802e")
	require.Contains(t, digests, "sha256:defe972d81b71f52a3c4fa30b5c6c192c950da7e937516d249429f7900d5f174")

}
