package github

import (
	"fmt"
	"testing"
)

var teamReposIngestor TeamReposIngestor

func init() {
	loadDataFromJSONFile("../../test/data/teams.json", &teamsIngestor.data)
	teamsIngestor.db = db
	teamsIngestor.Sync()

	loadDataFromJSONFile("../../test/data/team_repos.json", &teamReposIngestor.data)
	teamReposIngestor.db = db
	teamReposIngestor.teamSlug = "teama"
	teamReposIngestor.Sync()
}

func TestTeamReposInserted(t *testing.T) {
	var expectedRepos = []map[string]string{
		{"name": "repoA1", "team": "teama", "permission": "ADMIN"},
		{"name": "repoA2", "team": "teama", "permission": "WRITE"},
	}

	for _, expectedRepo := range expectedRepos {
		testname := fmt.Sprintf("%s,%s", expectedRepo["name"], expectedRepo["team"])
		records := db.Run(`
		MATCH (t:Team{slug: $team})-[rel:HAS_PERMISSION_ON]->(r:Repository{name: $name})
		RETURN rel.permission as permission
		`,
			map[string]interface{}{"team": expectedRepo["team"], "name": expectedRepo["name"]},
		)
		records.Next()

		permission, _ := records.Record().Get("permission")

		t.Run(testname, func(t *testing.T) {
			if permission != expectedRepo["permission"] {
				t.Errorf("got %s, want %s", permission, expectedRepo["permission"])
			}
		})
	}
}
