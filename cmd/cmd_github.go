package main

import (
	"flag"
	"fmt"

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
			"TeamMembers, Default (all previous), RepoWebhooks, OrganizationSecrets, "+
			"Environments, EnvironmentSecrets, Secrets (all GitHub secrets-related ingestors). "+
			"May be used multiple times.",
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
	gh := github.GetGitHub(db, githubApiURI, githubGraphQlURI, *githubToken, organization, session)
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

// Takes list of ingestor names, expands default, validates topics, and returns list of unique
// lowercase topics to ingest.
func resolveIngestorNames(names []string) ([]string, error) {
	names = sliceLower(names)
	validNames := []string{
		"organizations",
		"teams",
		"users",
		"repos",
		"teamrepos",
		"teammembers",
		"repowebhooks",
		"organizationsecrets",
		"environments",
		"environmentsecrets",
		"reposecrets",
	}
	defaultNames := []string{
		"organizations",
		"teams",
		"users",
		"repos",
		"teamrepos",
		"teammembers",
	}
	secretsNames := []string{
		"organizationsecrets",
		"environments",
		"environmentsecrets",
		"reposecrets",
	}

	// If no names were passed on CLI, we return default names
	if len(names) == 0 {
		return defaultNames, nil
	}

	// Expand default names
	if sliceContains(names, "default") {
		names = sliceRemove(names, "default")
		names = append(names, defaultNames...)
	}

	// Expand secrets names
	if sliceContains(names, "secrets") {
		names = sliceRemove(names, "secrets")
		names = append(names, secretsNames...)
	}

	// Validate all names
	for _, name := range names {
		if !sliceContains(validNames, name) {
			return nil, fmt.Errorf("invalid ingestor name %s", name)
		}
	}

	// Remove duplicates
	names = sliceDeduplicate(names)

	return names, nil
}
