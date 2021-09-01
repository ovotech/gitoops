package main

import (
	"flag"

	"os"

	"github.com/ovotech/gitoops/pkg/circleci"
	"github.com/ovotech/gitoops/pkg/database"
	log "github.com/sirupsen/logrus"
)

func cmdCircleCI(cmd *flag.FlagSet) {
	// Setup common params and parse command.
	// We have to do this here because we have a custom flag in the GitHub command.
	setupCommonFlags()
	cmd.Parse(os.Args[2:])
	validateCommonParams()
	initLogging()
	validateCircleCIParams()

	log.Infof("Running CircleCI ingestors")

	db := database.GetDB(neo4jURI, neo4jUser, neo4jPassword)

	cci := circleci.GetCircleCI(db, organization, *circleCICookie)
	cci.Sync()
}

func validateCircleCIParams() {
	requiredFlags := map[string]string{
		*circleCICookie: "-cookie",
	}

	for k, v := range requiredFlags {
		if k == "" {
			log.Fatalf("The %s flag is required. See help for more details.", v)
		}
	}
}
