package github

import (
	"fmt"
	"testing"
)

var teamsIngestor TeamsIngestor

func init() {
	loadDataFromJSONFile("../../test/data/teams.json", &teamsIngestor.data)
	teamsIngestor.db = db
	teamsIngestor.Sync()
}

func TestTeamsInserted(t *testing.T) {
	var expectedTeams = []map[string]string{
		{"slug": "teama"},
		{"slug": "teamb"},
	}

	for _, expectedTeam := range expectedTeams {
		testname := fmt.Sprintf("%s,%s", expectedTeam["name"], expectedTeam["slug"])
		records := db.Run(
			"MATCH (t:Team{slug: $slug}) RETURN t",
			map[string]interface{}{"slug": expectedTeam["slug"]},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}
