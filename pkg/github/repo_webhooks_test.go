package github

import (
	"fmt"
	"testing"
)

var repoWebhooksIngestor RepoWebhooksIngestor

func init() {
	loadDataFromJSONFile("../../test/data/repos.json", &reposIngestor.data)
	reposIngestor.db = db
	reposIngestor.Sync()

	loadDataFromJSONFile("../../test/data/repo_webhooks.json", &repoWebhooksIngestor.data)
	repoWebhooksIngestor.db = db
	repoWebhooksIngestor.repoName = "repoA1"
	repoWebhooksIngestor.Sync()
}

func TestRepoWebhooksInserted(t *testing.T) {
	var expectedWebhooks = []map[string]string{
		{"repoName": "repoA1", "host": "region.webhooks.aws"},
	}

	for _, expected := range expectedWebhooks {
		testname := fmt.Sprintf("%s,%s", expected["repoName"], expected["host"])
		records := db.Run(`
		MATCH (r:Repository{name: $repoName})-[rel:HAS_WEBHOOK]->(w:Webhook{host: $host})
		RETURN w.host as host
		`,
			map[string]interface{}{"repoName": expected["repoName"], "host": expected["host"]},
		)
		records.Next()

		t.Run(testname, func(t *testing.T) {
			if records.Record() == nil {
				t.Errorf("record not found")
			}
		})
	}
}
