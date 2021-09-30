package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/ovotech/gitoops/pkg/database"
	"github.com/ovotech/gitoops/pkg/github"
)

var (
	githubToken   = os.Getenv("GITHUB_TOKEN")
	githubApiURI  = "https://api.github.com"
	githubGraphQlURI = "https://api.github.com/graphql"
	neo4jURI      = "neo4j://localhost:7687"
	neo4jUser     = "neo4j"
	neo4jPassword = "neo4j" // ignore
	organization  = os.Getenv("GITHUB_ORGANIZATION")
	session       = "e2e"
	db            = database.GetDB(neo4jURI, neo4jUser, neo4jPassword)
	ingestors     = []string{
		"organizations",
		"teams",
		"users",
		"repos",
		"teamrepos",
		"teammembers",
		"repowebhooks",
		"organizationsecrets",
		"environments",
		"environmentsecrets",
		"reposecrets"}
)

type property struct {
	name  string
	value string
}

type node struct {
	label    string
	property property
}

type relationship struct {
	label string
}

type testCase struct {
	a node
	r relationship
	b node
}

// Returns a relationship string (for display purposes) and a Cypher query that matches for a direct
// relationship given by testCase.
// Note that this fuction is vulnerable to a database query injection, but we are only passing
// trusted arguments from our test cases.
func makeRelationshipQuery(tc testCase) (string, string) {
	relationship := fmt.Sprintf(
		`(:%s{%s:"%s"})-[:%s]->(:%s{%s:"%s"})`,
		tc.a.label,
		tc.a.property.name,
		tc.a.property.value,
		tc.r.label,
		tc.b.label,
		tc.b.property.name,
		tc.b.property.value,
	)
	query := fmt.Sprintf("MATCH p=%s RETURN p", relationship)
	return relationship, query
}

// Run a single test case.
func runTestCase(tc testCase, t *testing.T) {
	testName, query := makeRelationshipQuery(tc)

	records := db.Run(query, map[string]interface{}{})
	records.Next()

	t.Run(testName, func(t *testing.T) {
		if records.Record() == nil {
			t.Errorf("record not found")
		}
	})
}

func TestMain(m *testing.M) {
	gh := github.GetGitHub(db, githubApiURI, githubGraphQlURI, githubToken, organization, session)
	gh.SyncByIngestorNames(ingestors)
	exitVal := m.Run()
	os.Exit(exitVal)
}
