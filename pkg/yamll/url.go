package yamll

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
	"github.com/nikhilsbhat/yamll/pkg/errors"
)

// FetchData implements the methods to fetch the yaml data from various sources.
type FetchData interface {
	URL(log *slog.Logger) (string, error)
	Git(log *slog.Logger) (string, error)
	File(_ *slog.Logger) (string, error)
}

// URL reads the data from the URL import.
func (dependency *Dependency) URL(log *slog.Logger) (File, error) {
	httpClient := resty.New()

	var auth Auth
	if dependency.Auth != nil {
		auth = *dependency.Auth
	}

	if len(auth.BarerToken) != 0 {
		log.Debug("using token based auth for remote URL", slog.Any("url", dependency.Path))

		httpClient.SetAuthToken(auth.BarerToken)
	} else if len(auth.UserName) != 0 && len(auth.Password) != 0 {
		log.Debug("using basic auth for remote URL", slog.Any("url", dependency.Path))

		httpClient.SetBasicAuth(auth.UserName, auth.Password)
	}

	if len(auth.CaContent) != 0 {
		log.Debug("using CA for authentication for remote URL", slog.Any("url", dependency.Path))

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(auth.CaContent))
		httpClient.SetTLSClientConfig(&tls.Config{RootCAs: certPool}) //nolint:gosec
	} else {
		log.Debug("skipping TLS verification")

		httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	}

	resp, err := httpClient.R().Get(dependency.Path)
	if err != nil {
		return File{}, err
	}

	if resp.IsError() {
		return File{}, &errors.YamlError{
			Message: fmt.Sprintf("fetching URL '%s' failed with status %s", dependency.Path, resp.Status()),
		}
	}

	return File{Name: dependency.Path, Data: resp.String()}, err
}
