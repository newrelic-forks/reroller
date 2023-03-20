package docker

// OCI index schema
type IndexResp struct {
	MediaType     string      `json:"mediaType"`
	SchemaVersion int         `json:"schemaVersion"`
	Manifests     []Manifests `json:"manifests"`
}
type Platform struct {
	Architecture string `json:"architecture"`
	Os           string `json:"os"`
}
type Platform0 struct {
	Architecture string `json:"architecture"`
	Os           string `json:"os"`
	Variant      string `json:"variant"`
}
type Annotations struct {
	VndDockerReferenceDigest string `json:"vnd.docker.reference.digest"`
	VndDockerReferenceType   string `json:"vnd.docker.reference.type"`
}
type Manifests struct {
	MediaType   string      `json:"mediaType"`
	Digest      string      `json:"digest"`
	Size        int         `json:"size"`
	Platform    Platform    `json:"platform,omitempty"`
	Platform0   Platform0   `json:"platform,omitempty"`
	Annotations Annotations `json:"annotations,omitempty"`
}
