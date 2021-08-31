package circleci

import (
	"io"
	"net/http"
	"net/url"
	"path"

	gabs "github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

type RESTClient struct {
	client *http.Client
	cookie string
}

type RESTError struct {
	code    int
	message string
	count   int
}

// Retrieves a single page for a REST query.
func (c *RESTClient) call(resourcePath, pageToken string) (int, []byte) {
	log.Debugf("Issuing REST query for path %s", resourcePath)

	u, _ := url.Parse("https://circleci.com/api/v2/")
	u.Path = path.Join(u.Path, resourcePath)

	req, err := http.NewRequest(
		"GET",
		u.String(),
		nil,
	)
	if err != nil {
		panic(err)
	}

	req.AddCookie(&http.Cookie{
		Name:  "ring-session",
		Value: c.cookie,
	})
	req.Header.Add("accept", "application/json")

	if pageToken != "" {
		q := req.URL.Query()
		q.Add("page-token", pageToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return resp.StatusCode, body
}

// Retrieves all pages for a REST URL path.
func (c *RESTClient) fetch(resourcePath string, all bool) []byte {
	data := gabs.New()
	pageToken := ""

	// map status code to RESTError
	errorTracker := map[int]RESTError{}
	for {
		code, resp := c.call(resourcePath, pageToken)

		parsedResp, err := gabs.ParseJSON(resp)
		if err != nil {
			panic(err)
		}

		switch code {
		case 200:
			// All good, do nothing
		default:
			c.trackFetchErrors(code, parsedResp, errorTracker)
		}

		data.Merge(parsedResp)

		if !all || parsedResp.Path("next_page_token").Data() == nil {
			break
		}

		pageToken = parsedResp.Path("next_page_token").Data().(string)
	}

	c.logFetchErrors(resourcePath, errorTracker)

	return data.Bytes()
}

func (c *RESTClient) trackFetchErrors(
	code int,
	parsedResp *gabs.Container,
	errorTracker map[int]RESTError,
) {
	if _, ok := errorTracker[code]; ok {
		var resterror = errorTracker[code]
		resterror.count++
		errorTracker[code] = resterror
	} else {
		count := 1
		message := parsedResp.Path("message").Data().(string)
		errorTracker[code] = RESTError{
			code:    code,
			message: message,
			count:   count,
		}
	}
}

// Logs REST errors to output. These are logged as Warn because unlike GraphQL errors, we don't
// really expect any REST errors.
func (c *RESTClient) logFetchErrors(resourcePath string, errorTracker map[int]RESTError) {
	for _, e := range errorTracker {
		log.Warnf("%d errors on %s: %s", e.count, resourcePath, e.message)
	}
}
