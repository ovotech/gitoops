package github

import (
	"encoding/json"

	"github.com/ovotech/gitoops/pkg/database"
)

type TeamsIngestor struct {
	gqlclient *GraphQLClient
	db        *database.Database
	data      *TeamsData
}

type TeamsData struct {
	Edges []struct {
		Node struct {
			Name string `json:"name"`
			URL  string `json:"url"`
			Slug string `json:"slug"`
		} `json:"node"`
	} `json:"edges"`
}

func (ing *TeamsIngestor) FetchData() {
	query := `
	query($login: String!, $cursor: String) {
		organization(login: $login) {
			teams(first: 100, after: $cursor) {
				pageInfo {
					endCursor
					hasNextPage
				}
				edges {
					node {
						name
						url
						slug
					}
				}
				nodes {
					members {
						edges {
							role
						}
						nodes {
							url
							login
						}
					}
				}
			}
		}
	}
	`

	data := ing.gqlclient.fetch(
		query,
		"organization.teams",
		map[string]string{},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *TeamsIngestor) Sync() {
	ing.insertTeams()
}

func (ing *TeamsIngestor) insertTeams() {
	teams := []map[string]string{}

	for _, teamData := range ing.data.Edges {
		teams = append(teams, map[string]string{
			"url":  teamData.Node.URL,
			"name": teamData.Node.Name,
			"slug": teamData.Node.Slug,
		})
	}

	ing.db.Run(`
	UNWIND $teams AS team

	MERGE (t:Team{id: team.url})

	SET t.name = team.name,
	t.url = team.url,
	t.slug = team.slug
	`, map[string]interface{}{"teams": teams})
}
