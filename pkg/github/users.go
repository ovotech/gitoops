package github

import (
	"encoding/json"

	"github.com/ovotech/gitoops/pkg/database"
)

type UsersIngestor struct {
	gqlclient *GraphQLClient
	db        *database.Database
	data      *UsersData
}

type UsersData struct {
	Edges []struct {
		Role string `json:"role"`
	} `json:"edges"`
	Nodes []struct {
		Login string `json:"login"`
		URL   string `json:"url"`
	} `json:"nodes"`
}

func (ing *UsersIngestor) Sync() {
	ing.fetchData()
	ing.insertUsers()
}

func (ing *UsersIngestor) fetchData() {
	query := `
	query($login: String!, $cursor: String) {
		organization(login: $login) {
			membersWithRole(first: 100, after: $cursor) {
				pageInfo {
					endCursor
					hasNextPage
				}
				edges {
					role
				}
				nodes {
					login
					url
				}
			}
		}
	}
	`

	data := ing.gqlclient.fetch(
		query,
		"organization.membersWithRole",
		map[string]string{},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *UsersIngestor) insertUsers() {
	users := []map[string]string{}

	for i, userNode := range ing.data.Nodes {
		userEdge := ing.data.Edges[i]
		users = append(users, map[string]string{
			"url":          userNode.URL,
			"login":        userNode.Login,
			"role":         userEdge.Role,
			"organization": ing.gqlclient.organization,
		})
	}

	ing.db.Run(`
	UNWIND $users AS user

	MERGE (u:User{id: user.url})

	SET u.login = user.login

	WITH u, user

	MATCH (o:Organization{login: user.organization})
	MERGE (u)-[r:IS_MEMBER_OF{role: user.role}]->(o)
	`, map[string]interface{}{"users": users})
}
