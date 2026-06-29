package yamll

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikhilsbhat/yamll/pkg/errors"
)

type ociManifest struct {
	SchemaVersion int `json:"schemaVersion"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	} `json:"layers"`
}

type ociAuthChallenge struct {
	Realm   string
	Service string
	Scope   string
}

var ociHTTPClient = &http.Client{}

// SetOCIHTTPClientForTest overrides the OCI client used during tests.
func SetOCIHTTPClientForTest(client *http.Client) {
	if client == nil {
		ociHTTPClient = &http.Client{}

		return
	}

	ociHTTPClient = client
}

// OCI reads YAML content from an OCI artifact import.
func (dependency *Dependency) OCI(_ *slog.Logger) (File, error) {
	ref, err := parseOCIReference(dependency.Path)
	if err != nil {
		return File{}, err
	}

	var auth Auth
	if dependency.Auth != nil {
		auth = *dependency.Auth
	}

	manifestBody, err := ociGetWithAuth(
		ociHTTPClient,
		ociManifestURL(ref, ref.Reference),
		ref,
		auth,
	)
	if err != nil {
		return File{}, err
	}

	var manifest ociManifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		return File{}, &errors.YamllError{Message: fmt.Sprintf("reading OCI manifest errored with: %v", err)}
	}

	blobPayloads := make([][]byte, 0, len(manifest.Layers))

	for _, layer := range manifest.Layers {
		if layer.Digest == "" {
			continue
		}

		body, err := ociGetWithAuth(
			ociHTTPClient,
			ociBlobURL(ref, layer.Digest),
			ref,
			auth,
		)
		if err != nil {
			return File{}, err
		}

		blobPayloads = append(blobPayloads, body)
	}

	if len(blobPayloads) == 0 {
		return File{}, &errors.YamllError{Message: fmt.Sprintf("OCI artifact '%s' did not contain any readable layers", dependency.Path)}
	}

	var out bytes.Buffer

	for i, payload := range blobPayloads {
		if i > 0 {
			out.WriteString("\n---\n")
		}

		out.Write(bytes.TrimSpace(payload))
		out.WriteByte('\n')
	}

	sum := sha256.Sum256(out.Bytes())

	return File{
		Name: dependency.Path,
		Data: out.String(),
		Meta: FileMeta{SHA256: hex.EncodeToString(sum[:])},
	}, nil
}

type ociReference struct {
	Registry   string
	Repository string
	Reference  string
}

func parseOCIReference(raw string) (*ociReference, error) {
	trimmed := strings.TrimPrefix(raw, TypeOCI)
	if trimmed == "" {
		return nil, &errors.YamllError{Message: fmt.Sprintf("invalid OCI import: %q", raw)}
	}

	registry, repoRef, found := splitOCIRegistryAndRepo(trimmed)
	if !found || registry == "" || repoRef == "" {
		return nil, &errors.YamllError{Message: fmt.Sprintf("invalid OCI import reference: %q", raw)}
	}

	repo, ref, found := strings.Cut(repoRef, ":")
	if !found || repo == "" || ref == "" {
		return nil, &errors.YamllError{Message: fmt.Sprintf("invalid OCI import reference: %q", raw)}
	}

	return &ociReference{
		Registry:   registry,
		Repository: repo,
		Reference:  ref,
	}, nil
}

func ociGetWithAuth(client *http.Client, requestURL string, ref *ociReference, auth Auth) ([]byte, error) {
	body, challenge, err := ociFetch(client, requestURL, auth, "")
	if err == nil {
		return body, nil
	}

	if challenge == nil {
		return nil, err
	}

	token, err := ociToken(client, challenge, ref, auth)
	if err != nil {
		return nil, err
	}

	body, _, err = ociFetch(client, requestURL, auth, token)

	return body, err
}

func ociFetch(client *http.Client, requestURL string, auth Auth, bearer string) ([]byte, *ociAuthChallenge, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, nil, err
	}

	switch {
	case bearer != "":
		req.Header.Set("Authorization", "Bearer "+bearer)
	case auth.UserName != "" || auth.Password != "":
		req.SetBasicAuth(auth.UserName, auth.Password)
	case auth.BarerToken != "":
		req.Header.Set("Authorization", "Bearer "+auth.BarerToken)
	}

	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.artifact.manifest.v1+json",
	}, ", "))

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if challenge, ok := parseOCIChallenge(resp.Header.Get("WWW-Authenticate")); ok {
			return nil, &challenge, &errors.YamllError{Message: "oci registry requires auth"}
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)

		return nil, nil, &errors.YamllError{Message: ociRequestError(requestURL, resp.Status, payload)}
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return payload, nil, nil
}

func ociScheme(registry string) string {
	host := registry
	if h, _, err := net.SplitHostPort(registry); err == nil {
		host = h
	}

	if host == "localhost" || host == "127.0.0.1" {
		return "http"
	}

	return "https"
}

func ociManifestURL(ref *ociReference, reference string) string {
	return fmt.Sprintf("%s://%s/v2/%s/manifests/%s", ociScheme(ref.Registry), ref.Registry, ref.Repository, url.PathEscape(reference))
}

func ociBlobURL(ref *ociReference, digest string) string {
	return fmt.Sprintf("%s://%s/v2/%s/blobs/%s", ociScheme(ref.Registry), ref.Registry, ref.Repository, url.PathEscape(digest))
}

func splitOCIRegistryAndRepo(trimmed string) (string, string, bool) {
	slash := strings.LastIndex(trimmed, "/")
	if slash == -1 {
		return "", "", false
	}

	return trimmed[:slash], trimmed[slash+1:], true
}

func parseOCIChallenge(header string) (ociAuthChallenge, bool) {
	if !strings.HasPrefix(header, "Bearer ") {
		return ociAuthChallenge{}, false
	}

	challenge := ociAuthChallenge{}

	for part := range strings.SplitSeq(strings.TrimPrefix(header, "Bearer "), ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}

		value = strings.Trim(value, `"`)

		switch key {
		case "realm":
			challenge.Realm = value
		case "service":
			challenge.Service = value
		case "scope":
			challenge.Scope = value
		}
	}

	return challenge, challenge.Realm != ""
}

func ociToken(client *http.Client, challenge *ociAuthChallenge, ref *ociReference, auth Auth) (string, error) {
	tokenURL, err := url.Parse(challenge.Realm)
	if err != nil {
		return "", err
	}

	query := tokenURL.Query()

	if challenge.Service != "" {
		query.Set("service", challenge.Service)
	}

	if challenge.Scope != "" {
		query.Set("scope", challenge.Scope)
	} else {
		query.Set("scope", fmt.Sprintf("repository:%s:pull", ref.Repository))
	}

	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}

	switch {
	case auth.UserName != "" || auth.Password != "":
		req.SetBasicAuth(auth.UserName, auth.Password)
	case auth.BarerToken != "":
		req.Header.Set("Authorization", "Bearer "+auth.BarerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)

		return "", &errors.YamllError{Message: ociRequestError(tokenURL.String(), resp.Status, payload)}
	}

	var payload struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.Token != "" {
		return payload.Token, nil
	}

	return payload.AccessToken, nil
}

func ociRequestError(requestURL, status string, payload []byte) string {
	return fmt.Sprintf("oci request to %s failed with status %s: %s", requestURL, status, strings.TrimSpace(string(payload)))
}
