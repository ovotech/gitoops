package github

import (
	"fmt"
	"reflect"
	"testing"
)

var usersIngestor = UsersIngestor{
	gqlclient: gqlclient,
	db:        db,
	data:      &UsersData{},
}

func init() {
	loadDataFromJSONFile("../../test/data/users.json", &usersIngestor.data)
	usersIngestor.Sync()
}

func TestUsersInserted(t *testing.T) {
	var expectedUsers = []map[string]string{
		{"login": "userA", "organization": "fakenews", "role": "MEMBER"},
	}

	for _, expectedUser := range expectedUsers {
		testname := fmt.Sprintf(
			"%s,%s,%s",
			expectedUser["login"],
			expectedUser["organization"],
			expectedUser["role"],
		)

		records := db.Run(`
		MATCH (u:User{login: $login})-[r:IS_MEMBER_OF]->(o:Organization)
		RETURN u.login as login, r.role as role, o.login as organization
		`,
			map[string]interface{}{"login": expectedUser["login"]},
		)
		records.Next()

		login, _ := records.Record().Get("login")
		organization, _ := records.Record().Get("organization")
		role, _ := records.Record().Get("role")

		actualUser := map[string]string{
			"login":        login.(string),
			"organization": organization.(string),
			"role":         role.(string),
		}

		t.Run(testname, func(t *testing.T) {
			if !reflect.DeepEqual(actualUser, expectedUser) {
				t.Errorf("got %s, want %s", actualUser, expectedUser)
			}
		})
	}
}
