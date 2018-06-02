package checker

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/temoto/robotstxt"
)

const (
	maxRedirection = 100
	userAgent      = "FinalUrlBot 1.0"
)

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

// A Checker is an URL redirection checker client.
type Checker struct {
	CheckRobotTxt bool
}

// DefaultChecker is the default Checker and is used by GetRedirections
var DefaultChecker = &Checker{
	CheckRobotTxt: true,
}

// GetRedirections returns the redirections flow from dest to the final URL
// using the default Checker
func GetRedirections(dest string) (Flow, error) {
	return DefaultChecker.GetRedirections(dest)
}

// GetRedirections returns the redirections flow from dest to the final URL
func (c *Checker) GetRedirections(dest string) (Flow, error) {
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

		if c.CheckRobotTxt {
			isAllowed, err := isRobotsTxtAllowed(urlToProcess)
			if err != nil {
				return result, err
			}

			if !isAllowed {
				return result, errors.New("RobotTxt disabled crawling")
			}
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

func isRobotsTxtAllowed(urlToProcess string) (bool, error) {
	uri, err := url.ParseRequestURI(urlToProcess)
	if err != nil {
		return false, err
	}

	resp, err := http.Get(uri.Scheme + "://" + uri.Host + "/robots.txt")
	if err != nil {
		return false, nil
	}

	robots, err := robotstxt.FromResponse(resp)
	resp.Body.Close()
	if err != nil {
		return false, err
	}

	isAllowed := robots.TestAgent(urlToProcess, userAgent)

	return isAllowed, nil
}
