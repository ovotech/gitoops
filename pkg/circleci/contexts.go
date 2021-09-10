package circleci

import (
	"encoding/json"

	"github.com/ovotech/gitoops/pkg/database"
)

type ContextsIngestor struct {
	gqlclient      *GraphQLClient
	db             *database.Database
	data           *ContextsData
	organizationId string
	session        string
}

type ContextsData struct {
	Edges []struct {
		Node struct {
			Groups struct {
				Edges []struct {
					Node struct {
						Name string `json:"name"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"groups"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"node"`
	} `json:"edges"`
}

func (ing *ContextsIngestor) fetchData() {
	query := `
	query Contexts($orgId: ID!) {
		organization(id: $orgId) {
			contexts {
				edges {
					node {
						id
						name
						groups {
							edges {
								node {
									id
									name
								}
							}
						}
					}
				}
			}
		}
	}
	`

	data := ing.gqlclient.fetch(
		query,
		"organization.contexts",
		map[string]string{"orgId": ing.organizationId},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *ContextsIngestor) Sync() {
	ing.fetchData()
	ing.insertTeamsContexts()
	ing.insertAllMembersContexts()
}

// Insert contexts that are scoped to teams. This basically excludes contexts that "All members" can
// access.
func (ing *ContextsIngestor) insertTeamsContexts() {
	contexts := []map[string]interface{}{}

	for _, contextEdge := range ing.data.Edges {
		contextNode := contextEdge.Node
		for _, groupEdge := range contextNode.Groups.Edges {
			// we ignore "All members" here, that mapping is handled in another function
			if groupEdge.Node.Name == "All members" {
				continue
			}
			contexts = append(contexts, map[string]interface{}{
				"id":   contextNode.ID,
				"name": contextNode.Name,
				"team": groupEdge.Node.Name,
			})
		}
	}

	ing.db.Run(`
	UNWIND $contexts AS context

	MERGE (c:CircleCIContext{id: context.id})

	SET c.name = context.name,
	c.all_members = false,
	c.session = $session

	WITH c, context
	MATCH (t:Team{name: context.team})
	MERGE (t)-[rel:HAS_ACCESS_TO_CIRCLECI_CONTEXT]->(c)
	SET rel.session = $session
	`, map[string]interface{}{"contexts": contexts, "session": ing.session})
}

// Insert contexts that are accessible by all members of the GitHub org.
func (ing *ContextsIngestor) insertAllMembersContexts() {
	contexts := []map[string]interface{}{}

	for _, contextEdge := range ing.data.Edges {
		contextNode := contextEdge.Node
		for _, groupEdge := range contextNode.Groups.Edges {
			// we ignore anything that is not "All members" here
			if groupEdge.Node.Name != "All members" {
				continue
			}
			contexts = append(contexts, map[string]interface{}{
				"id":   contextNode.ID,
				"name": contextNode.Name,
			})
		}
	}

	ing.db.Run(`
	UNWIND $contexts AS context

	MERGE (c:CircleCIContext{id: context.id})

	SET c.name = context.name,
	c.all_members = true,
	c.session = $session

	WITH c, context
	MATCH (u:User)
	MERGE (u)-[rel:HAS_ACCESS_TO_CIRCLECI_CONTEXT]->(c)
	SET rel.session = $session
	`, map[string]interface{}{"contexts": contexts, "session": ing.session})
}
