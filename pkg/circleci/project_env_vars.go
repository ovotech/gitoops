package circleci

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/ovotech/gitoops/pkg/database"
)

// Creates a relationship for a repository that has a CircleCI project.
type ProjectEnvVarsIngestor struct {
	restclient   *RESTClient
	db           *database.Database
	data         *ProjectEnvVarsData
	organization string
	projectName  string
	session      string
}

type ProjectEnvVarsData struct {
	Items []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"items"`
}

func (ing *ProjectEnvVarsIngestor) fetchData() {
	query := fmt.Sprintf("project/gh/%s/%s/envvar", ing.organization, ing.projectName)
	data := ing.restclient.fetch(query, false)
	json.Unmarshal(data, &ing.data)
}

func (ing *ProjectEnvVarsIngestor) Sync() {
	ing.fetchData()
	ing.insertProjectEnvVars()
}

func (ing *ProjectEnvVarsIngestor) insertProjectEnvVars() {
	envVars := []map[string]interface{}{}

	for _, item := range ing.data.Items {
		id := fmt.Sprintf("%x", md5.Sum([]byte(ing.projectName+item.Name)))
		envVars = append(envVars, map[string]interface{}{
			"id":             id,
			"projectId":      ing.projectName,
			"name":           item.Name,
			"truncatedValue": item.Value[len(item.Value)-4:],
		})
	}

	ing.db.Run(`
	UNWIND $envVars AS envVar

	MERGE (v:EnvironmentVariable{id: envVar.id})

	SET v.name = envVar.name,
	v.session = $session

	WITH v, envVar
	MATCH (c:CircleCIProject{id: envVar.projectId})
	MERGE (c)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	SET rel.session = $session
	`, map[string]interface{}{"envVars": envVars, "session": ing.session})
}
