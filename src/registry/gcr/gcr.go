package gcr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	Registry = "gcr.io"
	BaseUrl  = "https://gcr.io/v2"
)

func GCRLikeImageInfo(baseUrl string) func(image, tag string) ([]string, error) {
	return func(image, tag string) ([]string, error) {
		resp, err := http.Get(fmt.Sprintf(strings.Trim(baseUrl, "/")+"/%s/tags/list", image))
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("gcr API returned %d", resp.StatusCode)
		}

		var partialresponse struct {
			Manifest map[string]struct {
				Tags []string `json:"tag"`
			} `json:"manifest"`
		}
		err = json.NewDecoder(resp.Body).Decode(&partialresponse)
		if err != nil {
			return nil, err
		}

		var digests []string

		for digest, data := range partialresponse.Manifest {
			for _, mTag := range data.Tags {
				if mTag == tag {
					digests = append(digests, digest)
				}
			}
		}

		return digests, nil
	}
}
