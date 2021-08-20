package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/notaryproject/hora/pkg/common"
	"github.com/notaryproject/hora/pkg/ocispecs"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

const unknownSize int64 = -1

func (c *Client) GetReferrers(ref common.Reference, artifactTypes []string, nextToken string) ([]ocispecs.ReferenceDescriptor, error) {
	resp, err := c.getReferrers(ref, artifactTypes, nextToken)

	if err != nil {
		return nil, err
	}

	if resp.Digest != ref.Digest.String() {
		// TODO %q versus %v
		return nil, fmt.Errorf("subject manifest mismatch. expected: %q got %q", ref.Digest, resp.Digest)
	}

	var refDescs []ocispecs.ReferenceDescriptor
	for _, manifest := range resp.Referrers {
		desc := ocispecs.ReferenceDescriptor{
			Descriptor: oci.Descriptor{
				MediaType: manifest.Manifest.MediaType,
				Digest:    digest.Digest(manifest.Digest),
				Size:      unknownSize,
			},
			ArtifactType: manifest.Manifest.ArtifactType,
		}
		refDescs = append(refDescs, desc)
	}

	return refDescs, nil

}

func (c *Client) getReferrers(ref common.Reference, _ []string, _ string) (*ocispecs.ReferrersResponse, error) {
	scheme := "https"
	if c.plainHTTP {
		scheme = "http"
	}

	refParts := strings.Split(ref.Path, "/")

	url := fmt.Sprintf("%s://%s/v2/_ext/oci-artifacts/v1/%s/manifests/%s/references",
		scheme,
		refParts[0],
		refParts[1],
		ref.Digest.String(),
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid reference: %v", ref.Original)
	}

	resp, err := c.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", url, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		// no op
	case http.StatusUnauthorized, http.StatusNotFound:
		return nil, fmt.Errorf("%v: %s", ref.Original, resp.Status)
	default:
		return nil, fmt.Errorf("%v: %s", ref.Original, resp.Status)
	}

	var referrersResp ocispecs.ReferrersResponse
	if err = json.NewDecoder(resp.Body).Decode(&referrersResp); err != nil {
		return nil, err
	}

	return &referrersResp, nil
}
