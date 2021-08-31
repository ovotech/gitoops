package circleci

import (
	"encoding/json"
)

type OrganizationIngestor struct {
	gqlclient *GraphQLClient
	// db           *database.Database
	data         *OrganizationData
	organization string
}

type OrganizationData struct {
	Id string `json:"id"`
}

func (ing *OrganizationIngestor) fetchData() {
	query := `
	query Organization($vcsType: VCSType!, $orgName: String!) {
		organization(vcsType: $vcsType, name: $orgName) {
			id
		}
	}
	`

	data := ing.gqlclient.fetch(
		query,
		"organization",
		map[string]string{"orgName": ing.organization},
	)

	json.Unmarshal(data, &ing.data)
}

// Returns organization ID
func (ing *OrganizationIngestor) GetOrganizationId() string {
	ing.fetchData()
	return ing.data.Id
}

// func (ing *OrganizationIngestor) Sync() {
// 	ing.fetchData()
// 	ing.insertOrganization()
// }

// func (ing *OrganizationIngestor) insertOrganization() {
// 	ing.db.Run(`
// 	MERGE (o:CircleCIOrganization{id: $id})
// 	`, map[string]interface{}{"id": ing.data.Id})
// }
