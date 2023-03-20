package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	Registry = "docker.io"
	BaseUrl  = "https://index.docker.io/v2"
	AuthUrl  = "https://auth.docker.io"
)

func DockerLikeImageInfo(baseUrl string) func(image, tag string) ([]string, error) {
	baseUrl = strings.Trim(baseUrl, "/")

	return func(image, tag string) ([]string, error) {
		manifestsUrl := fmt.Sprintf(baseUrl+"/%s/manifests/%s", image, tag)
		headRsp, err := http.Head(manifestsUrl)
		if err != nil {
			return nil, fmt.Errorf("requesting HEAD for %s: %v", manifestsUrl, err)
		}

		var token string
		wwwauthenticate := headRsp.Header.Get("www-authenticate")
		if wwwauthenticate != "" {
			au, err := authUrl(wwwauthenticate)
			if err != nil {
				return nil, fmt.Errorf("building auth url from header: %w", err)
			}

			token, err = authenticate(au)
			if err != nil {
				return nil, fmt.Errorf("requesting token from www-authenticate url: %w", err)
			}
		}

		req, err := http.NewRequest(http.MethodGet, manifestsUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("building manifest request: %w", err)
		}

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		digests := []string{}

		retDigests, err := ociDigests(req)
		if err != nil {
			return nil, fmt.Errorf("getting oci image digests from docker: %w", err)
		}

		digests = append(digests, retDigests...)

		// Query the manifest list digest
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
		retDigests, err = dockerContentDigest(req)
		if err != nil {
			return nil, fmt.Errorf("getting manifest list from docker: %w", err)
		}

		digests = append(digests, retDigests...)

		// Query the v2 manifest digest
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		retDigests, err = dockerContentDigest(req)
		if err != nil {
			return nil, fmt.Errorf("getting manifest from docker: %w", err)
		}

		digests = append(digests, retDigests...)

		if len(digests) == 0 {
			return nil, fmt.Errorf("getting images digests for the docker image %s:%s", image, tag)
		}

		return digests, nil
	}
}

func dockerContentDigest(req *http.Request) ([]string, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying dockerhub tags endpoint: %w", err)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	digest, found := resp.Header["Docker-Content-Digest"]
	if !found {
		// OCI images will not respond with the list of digest
		return nil, nil
	}

	return digest, nil
}

func ociDigests(req *http.Request) ([]string, error) {
	// https://github.com/moby/buildkit/issues/2251
	acceptHeader := "application/vnd.oci.image.index.v1+json"
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying dockerhub tags endpoint: %w", err)
	}

	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != acceptHeader {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	indexResp := &IndexResp{}

	if err := json.Unmarshal(body, indexResp); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	digests := []string{}

	for _, manifest := range indexResp.Manifests {
		digests = append(digests, manifest.Digest)
	}

	return digests, nil
}

func authUrl(wwwAuthenticate string) (*url.URL, error) {
	if !strings.HasPrefix(wwwAuthenticate, "Bearer ") {
		return nil, fmt.Errorf("www-authenticate header lacks Bearer prefix")
	}

	args := url.Values{}
	wwwAuthenticate = strings.TrimPrefix(wwwAuthenticate, "Bearer ")
	for _, entry := range strings.Split(wwwAuthenticate, ",") {
		entry = strings.TrimSpace(entry)

		keyval := strings.Split(entry, "=")
		if len(keyval) != 2 {
			return nil, fmt.Errorf("invalid keyval '%s'", keyval)
		}

		args.Set(keyval[0], strings.Trim(keyval[1], `"' `))
	}

	realm := args.Get("realm")
	service := args.Get("service")
	scope := args.Get("scope")
	if realm == "" || service == "" || scope == "" {
		return nil, fmt.Errorf("missing realm, service or scope in %v", args)
	}

	args.Del("realm")

	return url.Parse(realm + "?" + args.Encode())
}

func authenticate(authUrl *url.URL) (string, error) {
	req, err := http.NewRequest(http.MethodGet, authUrl.String(), nil)
	if err != nil {
		return "", fmt.Errorf("bulding auth request: %w", err)
	}

	authResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("retrieving token: %w", err)
	}
	if authResp.StatusCode >= 400 {
		return "", fmt.Errorf("dockerhub auth endpoint returned %d", authResp.StatusCode)
	}

	var authResponse struct {
		Token string `json:"token"`
	}

	err = json.NewDecoder(authResp.Body).Decode(&authResponse)
	if err != nil {
		return "", fmt.Errorf("decoding response from dockerhub auth endpoint: %w", err)
	}

	return authResponse.Token, nil
}
