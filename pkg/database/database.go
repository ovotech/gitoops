package database

import (
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Database struct {
	driver  neo4j.Driver
	session neo4j.Session
}

func GetDB(target, user, password string) *Database {
	driver, err := neo4j.NewDriver(target, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		log.Fatal(err)
	}

	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	return &Database{
		session: session,
	}
}

// Wipes the database
func (d *Database) Clear() {
	d.session.Run("MATCH (n) DETACH DELETE n", nil)
}

// Runs a single query, doesn't return results
func (d *Database) Run(query string, params map[string]interface{}) neo4j.Result {
	records, err := d.session.Run(query, params)
	if err != nil {
		panic(err)
	}

	return records
}

func (d *Database) Close() error {
	sessionErr := d.session.Close()
	driverErr := d.driver.Close()
	if driverErr == nil {
		return sessionErr
	}
	if sessionErr == nil {
		return driverErr
	}
	return fmt.Errorf("Both session and driver could not be closed."+
		"\nsession close failed with: %v"+
		"\ndriver close failed with: %v\n",
		sessionErr, driverErr)
}
