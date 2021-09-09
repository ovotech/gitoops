package main

import (
	"flag"

	"os"

	"github.com/ovotech/gitoops/pkg/database"
	"github.com/ovotech/gitoops/pkg/github"
	log "github.com/sirupsen/logrus"
)

func cmdGitHub(cmd *flag.FlagSet) {
	setupCommonFlags()
	// Since we want to support a list this has to be defined here, because Golang's `flag`
	// sucks (or more likely, I don't understand how to use it)
	githubCmd.Var(
		&githubIngestors,
		"ingestor",
		"Ingestors to call. Supports: Organizations, Teams, Users, Repos, TeamRepos, "+
			"TeamMembers, Default (all previous), RepoWebhooks. May be used multiple times.",
	)
	// Parse arguments
	cmd.Parse(os.Args[2:])
	validateCommonParams()
	initLogging()
	validateGitHubParams()

	log.Infof("Running GitHub ingestors")

	// Parse user submitted topics into a list of topics to ingest
	ingestorNames, err := resolveIngestorNames(githubIngestors)
	if err != nil {
		log.Fatalf("Error parsing topics: %s", err)
	}

	// Set up DB
	db := database.GetDB(neo4jURI, neo4jUser, neo4jPassword)

	// Now we can actually call the ingestor
	gh := github.GetGitHub(db, *githubToken, organization, session)
	gh.SyncByIngestorNames(ingestorNames)
}

func validateGitHubParams() {
	requiredFlags := map[string]string{
		*githubToken: "-token",
	}

	for k, v := range requiredFlags {
		if k == "" {
			log.Fatalf("The %s flag is required. See help for more details.", v)
		}
	}
}
