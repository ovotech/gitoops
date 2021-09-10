package circleci

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/ovotech/gitoops/pkg/database"
)

type ContextEnVarsIngestor struct {
	gqlclient *GraphQLClient
	db        *database.Database
	data      *ContextEnvVarsData
	contextId string
	session   string
}

type ContextEnvVarsData struct {
	Id        string `json:"id"`
	Resources []struct {
		Variable       string `json:"variable"`
		TruncatedValue string `json:"truncatedValue"`
	} `json:"resources"`
}

func (ing *ContextEnVarsIngestor) fetchData() {
	query := `
	query Context($contextId: ID!) {
		context(id: $contextId) {
			id
			resources {
				variable
				truncatedValue
			}
		}
	}	  
	`

	data := ing.gqlclient.fetch(
		query,
		"context",
		map[string]string{"contextId": ing.contextId},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *ContextEnVarsIngestor) Sync() {
	ing.fetchData()
	ing.insertContextsEnvVars()
}

func (ing *ContextEnVarsIngestor) insertContextsEnvVars() {
	envVars := []map[string]interface{}{}

	for _, resource := range ing.data.Resources {
		id := fmt.Sprintf("%x", md5.Sum([]byte(ing.data.Id+resource.Variable)))
		envVars = append(envVars, map[string]interface{}{
			"id":             id,
			"contextId":      ing.data.Id,
			"variable":       resource.Variable,
			"truncatedValue": resource.TruncatedValue,
		})
	}

	ing.db.Run(`
	UNWIND $envVars AS envVar

	MERGE (v:EnvironmentVariable{id: envVar.id})

	SET v.variable = envVar.variable,
	v.truncatedValue = envVar.truncatedValue

	WITH v, envVar
	MATCH (c:CircleCIContext{id: envVar.contextId})
	MERGE (c)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	`, map[string]interface{}{"envVars": envVars})
}
