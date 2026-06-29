package yamll_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
	"github.com/stretchr/testify/require"
)

func TestDependencyOCI(t *testing.T) {
	yamll.SetOCIHTTPClientForTest(&http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/manifests/v1"):
				body := strings.Join([]string{
					`{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:config","size":0},"layers":[`,
					`{"mediaType":"application/vnd.oci.image.layer.v1.tar+gzip","digest":"sha256:deadbeef","size":67}]}`,
				}, "")

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			case strings.HasSuffix(req.URL.Path, "/blobs/sha256:deadbeef"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: platform-config\n")),
					Header:     make(http.Header),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("not found")),
					Header:     make(http.Header),
				}, nil
			}
		}),
	})
	t.Cleanup(func() {
		yamll.SetOCIHTTPClientForTest(nil)
	})

	dependency := yamll.Dependency{
		Path: "oci://ghcr.io/company/platform-config:v1",
		Type: yamll.TypeOCI,
	}

	cfg := yamll.New(false, "DEBUG", "")
	cfg.SetLogger()

	out, err := dependency.ReadData(false, cfg.GetLogger())
	require.NoError(t, err)
	require.Equal(t, dependency.Path, out.Name)
	require.Contains(t, out.Data, "apiVersion: v1")
	require.Contains(t, out.Data, "kind: ConfigMap")
	require.Equal(t, 0, strings.Count(out.Data, "---"))
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
