package github

import (
	"fmt"
	"testing"
)

var reposIngestor = ReposIngestor{
	gqlclient: gqlclient,
	db:        db,
	data:      &ReposData{},
}

func init() {
	loadDataFromJSONFile("../../test/data/repos.json", &reposIngestor.data)
	reposIngestor.Sync()
}

func TestReposInserted(t *testing.T) {
	var expectedRepos = []map[string]string{
		{"name": "repoA1"},
	}

	for _, expectedRepo := range expectedRepos {
		testname := expectedRepo["name"]
		records := db.Run(`
		MATCH (r:Repository{name: $name})
		RETURN r
		`,
			map[string]interface{}{"name": expectedRepo["name"]},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}

func TestReposFilesInserted(t *testing.T) {
	var expectedReposFiles = []map[string]string{
		{"name": "repoA1", "path": ".circleci/config.yml"},
		{"name": "repoA1", "path": ".github/workflows/foo.yaml"},
	}

	for _, expectedRepoFile := range expectedReposFiles {
		testname := fmt.Sprintf("%s,%s", expectedRepoFile["name"], expectedRepoFile["path"])
		records := db.Run(`
		MATCH (r:Repository{name: $name})-[rel:HAS_CI_CONFIGURATION_FILE]->(f:File{path: $path})
		RETURN r
		`,
			map[string]interface{}{
				"name": expectedRepoFile["name"],
				"path": expectedRepoFile["path"],
			},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}

func TestReposCollaboratorsInserted(t *testing.T) {
	var expectedReposCollaborators = []map[string]string{
		{"name": "repoA1", "permission": "ADMIN", "login": "collaboratora1"},
	}

	for _, expectedRepoCollaborator := range expectedReposCollaborators {
		testname := fmt.Sprintf(
			"%s,%s,%s",
			expectedRepoCollaborator["name"],
			expectedRepoCollaborator["permission"],
			expectedRepoCollaborator["login"],
		)
		records := db.Run(`
		MATCH (r:Repository{name: $name})<-[rel:HAS_PERMISSION_ON{permission:$permission}]-(u:User{login: $login})
		RETURN r
		`,
			map[string]interface{}{
				"name":       expectedRepoCollaborator["name"],
				"permission": expectedRepoCollaborator["permission"],
				"login":      expectedRepoCollaborator["login"],
			},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}

func TestReposPullRequestsStatusChecksInserted(t *testing.T) {
	var expectedReposStatusChecks = []map[string]string{
		{"name": "repoA1", "host": "circleci.com"},
	}

	for _, expectedRepoStatusCheck := range expectedReposStatusChecks {
		testname := fmt.Sprintf(
			"%s,%s",
			expectedRepoStatusCheck["name"],
			expectedRepoStatusCheck["host"],
		)
		records := db.Run(`
		MATCH (r:Repository{name: $name})-[rel:HAS_STATUS_CHECK{context: "pull request"}]->(s:StatusCheck{host: $host})
		RETURN r
		`,
			map[string]interface{}{
				"name": expectedRepoStatusCheck["name"],
				"host": expectedRepoStatusCheck["host"],
			},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}

func TestReposDefaultBranchStatusChecksInserted(t *testing.T) {
	var expectedReposStatusChecks = []map[string]string{
		{"name": "repoA1", "host": "circleci.com"},
	}

	for _, expectedRepoStatusCheck := range expectedReposStatusChecks {
		testname := fmt.Sprintf(
			"%s,%s",
			expectedRepoStatusCheck["name"],
			expectedRepoStatusCheck["host"],
		)
		records := db.Run(`
		MATCH (r:Repository{name: $name})-[rel:HAS_STATUS_CHECK{context: "default branch"}]->(s:StatusCheck{host: $host})
		RETURN r
		`,
			map[string]interface{}{
				"name": expectedRepoStatusCheck["name"],
				"host": expectedRepoStatusCheck["host"],
			},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}

func TestReposBranchProtectionRulesInserted(t *testing.T) {
	var expectedBranchProtectionRules = []map[string]string{
		{"name": "repoA1", "pattern": "master"},
	}

	for _, expected := range expectedBranchProtectionRules {
		testname := expected["pattern"]
		records := db.Run(`
		MATCH (r:Repository{name: $name})-[rel:HAS_BRANCH_PROTECTION_RULE]->(s:BranchProtectionRule{pattern: $pattern})
		RETURN r
		`,
			map[string]interface{}{"name": expected["name"], "pattern": expected["pattern"]},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}
