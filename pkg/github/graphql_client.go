package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	gabs "github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

var (
	fatalErrors = []string{
		"Resource protected by organization SAML enforcement",
		"Your token has not been granted the required scopes to execute this",
	}
)

type GraphQLClient struct {
	client           *http.Client
	token            string
	organization     string
	githubGraphQLURL string
}

type GraphQLError struct {
	errorType string
	message   string
	count     int
}

// Retrieves a single page for the GraphQL query.
func (c *GraphQLClient) call(query string, variables map[string]string) []byte {
	jsonValue, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("Issuing GraphQL query: %v", string(jsonValue))

	req, err := http.NewRequest(
		"POST",
		c.githubGraphQLURL,
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		log.Panic(err)
	}

	req.Header.Add("Authorization", "token "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	switch resp.StatusCode {
	case 200:
		// All good, do nothing
	case 502:
		log.Panic(
			"Received a 502 from GraphQL API. Sometimes this happens when we query too many resources at once (try lowering the 'first' in GraphQL query)",
		)
	default:
		log.Panicf(
			"Received HTTP status code %d from GraphQL API.",
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	return body
}

// Retrieves all pages for the GraphQL query.
func (c *GraphQLClient) fetch(query, resourcePath string, variables map[string]string) []byte {
	hasNextPagePath := fmt.Sprintf("data.%s.pageInfo.hasNextPage", resourcePath)
	cursorPath := fmt.Sprintf("data.%s.pageInfo.endCursor", resourcePath)
	dataPath := fmt.Sprintf("data.%s", resourcePath)

	variables["login"] = c.organization

	data := gabs.Container{}
	errorTracker := map[string]GraphQLError{}
	for {
		resp := c.call(query, variables)

		parsedResp, err := gabs.ParseJSON(resp)
		if err != nil {
			log.Panic(err)
		}

		// track GraphQL errors for diagnostics
		gqlerrors := parsedResp.Path("errors")
		if gqlerrors != nil {
			c.checkFatalErrors(gqlerrors)
			c.trackFetchErrors(gqlerrors, errorTracker)
		}

		// Merge this page's data into all data
		d := parsedResp.Path(dataPath)
		data.Merge(d)

		// Handle pagination
		hasNextPageData := parsedResp.Path(hasNextPagePath).Data()
		if hasNextPageData == nil {
			log.Warnf(
				"No hasNextPage in pageInfo for %s, something is wrong",
				resourcePath,
			)
			break
		}
		hasNextPage := hasNextPageData.(bool)
		if !hasNextPage {
			break
		}

		cursor := parsedResp.Path(cursorPath).Data().(string)
		variables["cursor"] = cursor
	}

	c.logFetchErrors(resourcePath, errorTracker)

	return data.Bytes()
}

// Takes errors container received from a GraphQL JSON response and updates the errorTracker.
func (c *GraphQLClient) trackFetchErrors(
	errors *gabs.Container,
	errorTracker map[string]GraphQLError,
) {
	for _, e := range errors.Children() {
		message := e.Path("message").Data().(string)
		if _, ok := errorTracker[message]; ok {
			var gqlerror = errorTracker[message]
			gqlerror.count++
			errorTracker[message] = gqlerror
		} else {
			count := 1
			errorType := e.Path("type").Data().(string)
			errorTracker[message] = GraphQLError{
				errorType: errorType,
				message:   message,
				count:     count,
			}
		}
	}
}

// Checks error container received from a GraphQL JSON response for fatal errors that should stop
// execution.
func (c *GraphQLClient) checkFatalErrors(errors *gabs.Container) {
	for _, e := range errors.Children() {
		message := e.Path("message").Data().(string)
		for _, fatalError := range fatalErrors {
			if strings.Contains(message, fatalError) {
				errorType := e.Path("type").Data().(string)
				log.Fatalf("Fatal GraphQL error received from GitHub: %s %s", errorType, message)
			}
		}
	}
}

// Logs GraphQL errors to output. Some amount of errors are expected even
// with full scopes (at a minimum, some FORBIDDEN errors when listing collaborators for repos, see:
// https://github.community/t/list-collaborators-api-v4/13571)
func (c *GraphQLClient) logFetchErrors(resourcePath string, errorTracker map[string]GraphQLError) {
	for _, e := range errorTracker {
		log.Warnf("%d errors on %s: %s %s", e.count, resourcePath, e.errorType, e.message)
	}
}
