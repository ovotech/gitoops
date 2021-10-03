package github

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	gabs "github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

type RESTClient struct {
	client        *http.Client
	token         string
	organization  string
	githubRESTURL string
}

type RESTError struct {
	code    int
	message string
	count   int
}

// Retrieves a single page for a REST query.
func (c *RESTClient) call(resourcePath string, page int) (int, []byte) {
	log.Debugf("Issuing REST query %s page %d", resourcePath, page)

	u, _ := url.Parse(c.githubRESTURL)
	u.Path = path.Join(u.Path, resourcePath)
	req, err := http.NewRequest(
		"GET",
		u.String(),
		nil,
	)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", "token "+c.token)

	q := req.URL.Query()
	q.Add("page", fmt.Sprint(page))
	q.Add("per_page", "100")
	req.URL.RawQuery = q.Encode()

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
func (c *RESTClient) fetch(resourcePath string) []byte {
	data := gabs.New()
	page := 1

	// map status code to RESTError
	errorTracker := map[int]RESTError{}
	// some APIs (like repo webhooks) return a list of objects as the root level element
	// others return an object with a top level "total_count" along with list of objects
	// we treat each case separately and use isObjectAPI to track which one we're dealing with
	isObjectAPI := false

	for {
		code, resp := c.call(resourcePath, page)

		parsedResp, err := gabs.ParseJSON(resp)
		if err != nil {
			panic(err)
		}

		if code != 200 {
			c.trackFetchErrors(code, parsedResp, errorTracker)
		}

		// note: we're currently running this check on each iteration even though we expect it to
		// be the same for all iterations of this loop. we have no way of knowing ahead of time
		// whether an API will return list or an object at root level (unless we hardcoded it)
		totalCount, ok := parsedResp.Path("total_count").Data().(float64)
		if ok {
			// handle cases like {"total_count": 0, "objects": []}
			isObjectAPI = true

			data.Merge(parsedResp)

			if page*100 > int(totalCount) {
				break
			}
		} else {
			// handle cases like [{}, {}, ...]
			// note that gabs doesn't support merging two arrays if they are the root element
			// see: https://github.com/Jeffail/gabs/issues/60
			// the recommended workaround is to place the arrays in a field before merging
			d := gabs.New()
			d.Array("nodes")
			d.Set(parsedResp, "nodes")

			data.Merge(d)

			// If we have less than 100 items on a page, we've reached the end.
			count, _ := parsedResp.ArrayCount()
			if count < 100 {
				break
			}
		}

		page += 1
	}

	c.logFetchErrors(resourcePath, errorTracker)

	if isObjectAPI {
		return data.Bytes()
	}

	return data.Search("nodes").Bytes()
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
