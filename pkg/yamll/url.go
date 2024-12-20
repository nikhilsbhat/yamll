package yamll

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"

	"github.com/go-resty/resty/v2"
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

	if len(dependency.Auth.BarerToken) != 0 {
		log.Debug("using token based auth for remote URL", slog.Any("url", dependency.Path))

		httpClient.SetAuthToken(dependency.Auth.BarerToken)
	} else if len(dependency.Auth.UserName) != 0 && len(dependency.Auth.Password) != 0 {
		log.Debug("using basic auth for remote URL", slog.Any("url", dependency.Path))

		httpClient.SetBasicAuth(dependency.Auth.UserName, dependency.Auth.Password)
	}

	if len(dependency.Auth.CaContent) != 0 {
		log.Debug("using CA for authentication for remote URL", slog.Any("url", dependency.Path))

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(dependency.Auth.CaContent))
		httpClient.SetTLSClientConfig(&tls.Config{RootCAs: certPool}) //nolint:gosec
	} else {
		log.Debug("skipping TLS verification")

		httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	}

	resp, err := httpClient.R().Get(dependency.Path)
	if err != nil {
		return File{}, err
	}

	return File{Name: dependency.Path, Data: resp.String()}, err
}
