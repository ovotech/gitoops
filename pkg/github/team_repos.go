package github

import (
	"encoding/json"

	"github.com/ovotech/gitoops/pkg/database"
)

type TeamReposIngestor struct {
	gqlclient *GraphQLClient
	db        *database.Database
	data      *TeamReposData
	teamSlug  string
}

type TeamReposData struct {
	Edges []struct {
		Permission string `json:"permission"`
	} `json:"edges"`
	Nodes []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"nodes"`
}

func (ing *TeamReposIngestor) Sync() {
	ing.fetchData()
	ing.insertTeamRepos()
}

func (ing *TeamReposIngestor) fetchData() {
	query := `
	query($login: String!, $teamSlug: String!, $cursor: String)  {
		organization(login: $login) {
			team(slug: $teamSlug) {
				repositories(first: 100, after: $cursor) {
					pageInfo {
						endCursor
						hasNextPage
					}
					edges {
						permission
					}
					nodes {
						url
						name
					}
				}
			}
		}
	}
	`

	data := ing.gqlclient.fetch(
		query,
		"organization.team.repositories",
		map[string]string{"teamSlug": ing.teamSlug},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *TeamReposIngestor) insertTeamRepos() {
	repos := []map[string]string{}

	for i, repoEdge := range ing.data.Edges {
		repoNode := ing.data.Nodes[i]
		repos = append(repos, map[string]string{
			"url":        repoNode.URL,
			"name":       repoNode.Name,
			"teamSlug":   ing.teamSlug,
			"permission": repoEdge.Permission,
		})
	}

	ing.db.Run(`
	UNWIND $repos as repo

	MERGE (r:Repository{id: repo.url})

	SET r.url = repo.url,
	r.name = repo.name

	WITH r, repo

	MATCH (t:Team{slug: repo.teamSlug})
	MERGE (t)-[rel:HAS_PERMISSION_ON{permission: repo.permission}]->(r)
	`, map[string]interface{}{"repos": repos})
}
