package github

import (
	"fmt"
	"testing"
)

var teamMembersIngestor TeamMembersIngestor

func init() {
	loadDataFromJSONFile("../../test/data/teams.json", &teamsIngestor.data)
	teamsIngestor.db = db
	teamsIngestor.Sync()

	loadDataFromJSONFile("../../test/data/team_members.json", &teamMembersIngestor.data)
	teamMembersIngestor.db = db
	teamMembersIngestor.teamSlug = "teama"
	teamMembersIngestor.Sync()
}

func TestTeamMembersInserted(t *testing.T) {
	var expectedMembers = []map[string]string{
		{"team": "teama", "login": "userA", "role": "MEMBER"},
	}

	for _, expected := range expectedMembers {
		testname := fmt.Sprintf("%s,%s", expected["team"], expected["login"])
		records := db.Run(`
		MATCH (t:Team{slug: $team})<-[rel:IS_MEMBER_OF]-(u:User{login: $login})
		RETURN u
		`,
			map[string]interface{}{"team": expected["team"], "login": expected["login"]},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}
