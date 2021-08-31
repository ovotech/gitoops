package circleci

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
		"Something unexpected happened.",
		"No value was provided for variable `vcsType', which is non-nullable.",
		"Non-nullable field was null.",
	}
)

type GraphQLClient struct {
	client *http.Client
	cookie string
}

type GraphQLError struct {
	message string
	count   int
}

// Makes a GraphQL call and returns full body
func (c *GraphQLClient) call(query string, variables map[string]string) []byte {
	variables["vcsType"] = "GITHUB"

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
		"https://circleci.com/graphql-unstable",
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		log.Panic(err)
	}

	req.AddCookie(&http.Cookie{
		Name:  "ring-session",
		Value: c.cookie,
	})
	req.Header.Add("content-type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	switch resp.StatusCode {
	case 200:
		// All good, do nothing
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

// Issues a GraphQL query and returns data in resourcePath.
func (c *GraphQLClient) fetch(query, resourcePath string, variables map[string]string) []byte {
	dataPath := fmt.Sprintf("data.%s", resourcePath)

	resp := c.call(query, variables)

	parsedResp, err := gabs.ParseJSON(resp)
	if err != nil {
		log.Panic(err)
	}

	gqlerrors := parsedResp.Path("errors")
	if gqlerrors != nil {
		c.checkFatalErrors(gqlerrors)
		errorTracker := map[string]GraphQLError{}
		c.trackFetchErrors(gqlerrors, errorTracker)
		c.logFetchErrors(resourcePath, errorTracker)
	}

	data := parsedResp.Path(dataPath)
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
			errorTracker[message] = GraphQLError{
				message: message,
				count:   count,
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
				log.Fatalf(
					"Fatal GraphQL error received from CircleCI: %s",
					message,
				)
			}
		}
	}
}

// Logs GraphQL errors to output. Some amount of errors are expected even
// with full scopes (at a minimum, some FORBIDDEN errors when listing collaborators for repos, see:
// https://github.community/t/list-collaborators-api-v4/13571)
func (c *GraphQLClient) logFetchErrors(resourcePath string, errorTracker map[string]GraphQLError) {
	for _, e := range errorTracker {
		log.Warnf("%d errors on %s: %s", e.count, resourcePath, e.message)
	}
}
