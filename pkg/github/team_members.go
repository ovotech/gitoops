package github

import (
	"encoding/json"

	"github.com/ovotech/gitoops/pkg/database"
)

// Ingests team members. We use a separate ingestor for this instead of the teams ingestor because
// we could have a team with more than 100 members and have to iterate over pages.
type TeamMembersIngestor struct {
	gqlclient *GraphQLClient
	db        *database.Database
	data      *TeamMembersData
	teamSlug  string
}

type TeamMembersData struct {
	Edges []struct {
		Role string `json:"role"`
	} `json:"edges"`
	Nodes []struct {
		URL   string `json:"url"`
		Login string `json:"login"`
	} `json:"nodes"`
}

func (ing *TeamMembersIngestor) FetchData() {
	query := `
	query ($login: String!, $teamSlug: String!, $cursor: String) {
		organization(login: $login) {
			team(slug: $teamSlug) {
				members(after: $cursor) {
					pageInfo {
						endCursor
						hasNextPage
					}
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
	`

	data := ing.gqlclient.fetch(
		query,
		"organization.team.members",
		map[string]string{"teamSlug": ing.teamSlug},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *TeamMembersIngestor) Sync() {
	ing.insertTeamMembers()
}

func (ing *TeamMembersIngestor) insertTeamMembers() {
	members := []map[string]string{}

	for i, teamMemberEdge := range ing.data.Edges {
		teamMemberNode := ing.data.Nodes[i]

		members = append(members, map[string]string{
			"url":      teamMemberNode.URL,
			"login":    teamMemberNode.Login,
			"role":     teamMemberEdge.Role,
			"teamSlug": ing.teamSlug,
		})
	}

	ing.db.Run(`
	UNWIND $members as member

	MERGE (u:User{id: member.url})

	SET u.login = member.login,
	u.url = member.url

	WITH u, member

	MATCH (t:Team{slug: member.teamSlug})
	MERGE (u)-[rel:IS_MEMBER_OF{role: member.role}]->(t)
	`, map[string]interface{}{"members": members})
}
