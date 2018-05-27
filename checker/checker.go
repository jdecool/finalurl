package checker

import (
	"errors"
	"net/http"
	"net/url"
)

const maxRedirection = 100

// Response represents HTTP response
type Response struct {
	URL        *url.URL
	StatusCode int
}

// Flow represents consecutive requests to the final URL
type Flow struct {
	OriginalURL   string
	Redirections  []Response
	FinalResponse *Response
}

// GetRedirections returns the redirections flow from destto go to the final URL
func GetRedirections(dest string) (Flow, error) {
	result := Flow{
		OriginalURL:   dest,
		Redirections:  []Response{},
		FinalResponse: nil,
	}

	urlToProcess := dest

	for {
		if len(result.Redirections) >= maxRedirection {
			break
		}

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}}

		uri, err := url.ParseRequestURI(urlToProcess)
		if err != nil {
			return result, err
		}

		if !uri.IsAbs() {
			redirectionLength := len(result.Redirections) - 1
			if redirectionLength == 0 {
				return result, errors.New("URL is not valid")
			}

			lastResponse := result.Redirections[len(result.Redirections)-1]
			lastURI, err := url.ParseRequestURI(lastResponse.URL.String())
			if err != nil {
				return result, err
			}

			urlToProcess = lastURI.Scheme + "://" + lastURI.Hostname() + urlToProcess
		}

		resp, err := client.Get(urlToProcess)
		if err != nil {
			return result, err
		}

		currentResponse := Response{
			URL:        resp.Request.URL,
			StatusCode: resp.StatusCode,
		}

		if currentResponse.StatusCode < 300 || currentResponse.StatusCode >= 400 {
			result.FinalResponse = &currentResponse
			break
		}

		result.Redirections = append(result.Redirections, currentResponse)

		urlToProcess = resp.Header.Get("Location")
	}

	return result, nil
}
