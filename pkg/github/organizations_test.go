package github

import (
	"fmt"
	"testing"
)

var orgIngestor OrganizationsIngestor

func init() {
	loadDataFromJSONFile("../../test/data/organizations.json", &orgIngestor.data)
	orgIngestor.db = db
	orgIngestor.Sync()
}

func TestOrgsInserted(t *testing.T) {
	var expectedOrgs = []map[string]string{
		{"login": "fakenews"},
	}

	for _, expectedOrg := range expectedOrgs {
		testname := fmt.Sprintf("%s,%s", expectedOrg["login"], expectedOrg["url"])
		records := db.Run(
			"MATCH (o:Organization{login: $organization}) RETURN o",
			map[string]interface{}{"organization": expectedOrg["login"]},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}
