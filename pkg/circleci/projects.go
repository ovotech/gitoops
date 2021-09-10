package circleci

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovotech/gitoops/pkg/database"
)

// Creates a relationship for a repository that has a CircleCI project.
type ProjectIngestor struct {
	restclient   *RESTClient
	db           *database.Database
	data         *ProjectData
	organization string
	repoName     string
	session      string
}

type ProjectData struct {
	Items []struct {
		ID          string        `json:"id"`
		Errors      []interface{} `json:"errors"`
		ProjectSlug string        `json:"project_slug"`
		UpdatedAt   time.Time     `json:"updated_at"`
		Number      int           `json:"number"`
		State       string        `json:"state"`
		CreatedAt   time.Time     `json:"created_at"`
		Trigger     struct {
			ReceivedAt time.Time `json:"received_at"`
			Type       string    `json:"type"`
			Actor      struct {
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
			} `json:"actor"`
		} `json:"trigger"`
		Vcs struct {
			OriginRepositoryURL string `json:"origin_repository_url"`
			TargetRepositoryURL string `json:"target_repository_url"`
			Revision            string `json:"revision"`
			ProviderName        string `json:"provider_name"`
			Commit              struct {
				Body    string `json:"body"`
				Subject string `json:"subject"`
			} `json:"commit"`
			Branch string `json:"branch"`
		} `json:"vcs"`
	} `json:"items"`
}

func (ing *ProjectIngestor) fetchData() {
	query := fmt.Sprintf("project/gh/%s/%s/pipeline", ing.organization, ing.repoName)
	data := ing.restclient.fetch(query, false)
	json.Unmarshal(data, &ing.data)
}

func (ing *ProjectIngestor) Sync() {
	ing.fetchData()
	ing.insertProject()
}

func (ing *ProjectIngestor) insertProject() {
	if len(ing.data.Items) < 1 {
		return
	}

	ing.db.Run(`
	MERGE (p:CircleCIProject{id: $repoName})

	SET p.repository = $repoName
	p.session = $session

	WITH p, $repoName as repoName

	MATCH (r:Repository{name: $repoName})
	MERGE (r)-[rel:HAS_CI]->(p)
	SET rel.session = $session
	`, map[string]interface{}{"repoName": ing.repoName, "session": ing.session})
}
