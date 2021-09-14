package github

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/ovotech/gitoops/pkg/database"
)

type EnvironmentSecretsIngestor struct {
	restclient *RESTClient
	db         *database.Database
	data       *EnvironmentSecretsData
	repoId     int64
	envName    string
	session    string
}

type EnvironmentSecretsData struct {
	Secrets []struct {
		Name string `json:"name"`
	} `json:"secrets"`
}

func (ing *EnvironmentSecretsIngestor) Sync() {
	ing.fetchData()
	ing.insertEnvironmentVariables()
}

func (ing *EnvironmentSecretsIngestor) fetchData() {
	query := fmt.Sprintf("repositories/%d/environments/%s/secrets", ing.repoId, ing.envName)

	data := ing.restclient.fetch(query)
	json.Unmarshal(data, &ing.data)
}

func (ing *EnvironmentSecretsIngestor) insertEnvironmentVariables() {
	envVars := []map[string]interface{}{}

	for _, envVar := range ing.data.Secrets {
		strRepoId := fmt.Sprintf("%d", ing.repoId)
		id := fmt.Sprintf("%x", md5.Sum([]byte(strRepoId+ing.envName+envVar.Name)))
		envVars = append(envVars, map[string]interface{}{
			"id":      id,
			"name":    envVar.Name,
			"repoId":  ing.repoId,
			"envName": ing.envName,
		})
	}

	ing.db.Run(`
	UNWIND $envVars AS envVar

	MERGE (v:EnvironmentVariable{id: envVar.id})

	SET v.name = envVar.name,
	v.session = $session

	WITH v, envVar

	MATCH (:Repository{databaseId: envVar.repoId})-->(e:Environment{name: envVar.envName})
	MERGE (e)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	SET rel.session = $session
	`, map[string]interface{}{"envVars": envVars, "session": ing.session})
}
