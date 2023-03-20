package quay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	Registry = "quay.io"
	BaseUrl  = "https://quay.io/api/v1/repository"
)

func QuayLikeImageInfo(baseUrl string) func(image, tag string) ([]string, error) {
	return func(image, tag string) ([]string, error) {
		resp, err := http.Get(fmt.Sprintf(strings.Trim(baseUrl, "/")+"/%s/tag/?specificTag=%s", image, tag))
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("quay API returned %d", resp.StatusCode)
		}

		var partialresponse struct {
			Tags []struct {
				Digest     string `json:"manifest_digest"`
				IsManifest bool   `json:"is_manifest_list"`
			} `json:"tags"`
		}
		err = json.NewDecoder(resp.Body).Decode(&partialresponse)
		if err != nil {
			return nil, err
		}

		if len(partialresponse.Tags) < 1 {
			return nil, fmt.Errorf("quay API did not return any image")
		}

		var digests []string
		for _, tag := range partialresponse.Tags {
			digests = append(digests, tag.Digest)
			if !tag.IsManifest {
				break
			}
		}

		return digests, nil
	}
}
