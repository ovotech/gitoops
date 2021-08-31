package github

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ovotech/gitoops/pkg/database"
)

const organization = "fakenews"

var db = func() *database.Database {
	neo4jPassword := os.Getenv("NEO4J_PASSWORD")
	db := database.GetDB("neo4j://localhost:7687", "neo4j", neo4jPassword)
	// clear db before tests
	db.Run(`
	MATCH (o:Organization{login: $organization})<-[]-(n)
	DETACH DELETE o,n
		`, map[string]interface{}{"organization": organization})
	return db
}()

var gqlclient = func() *GraphQLClient {
	return &GraphQLClient{
		client:       nil,
		token:        "",
		organization: organization,
	}
}()

func TestMain(m *testing.M) {
	exitVal := m.Run()
	// clear the test db after all tests
	db.Run(`
	MATCH (o:Organization{login: $organization})<-[]-(n)
	DETACH DELETE o,n
		`, map[string]interface{}{"organization": organization})
	os.Exit(exitVal)
}

func loadDataFromJSONFile(path string, i interface{}) {
	if data, err := ioutil.ReadFile(path); err == nil {
		json.Unmarshal(data, &i)
	} else {
		panic(err)
	}
}
