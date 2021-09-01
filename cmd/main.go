package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ovotech/gitoops/pkg/database"
	"github.com/ovotech/gitoops/pkg/enrich"
	log "github.com/sirupsen/logrus"
)

var (
	// Common parameters for all commands
	debug         bool
	organization  string
	neo4jURI      string
	neo4jUser     string
	neo4jPassword string

	githubCmd       = flag.NewFlagSet("github", flag.ExitOnError)
	githubToken     = githubCmd.String("token", "", "The GitHub access token.")
	githubIngestors arrayFlags

	circleCICmd    = flag.NewFlagSet("circleci", flag.ExitOnError)
	circleCICookie = circleCICmd.String(
		"cookie",
		"",
		"The 'ring-session' cookie from a CircleCI browser session. Get this from the network tab as you're browsing the CircleCI app authenticated.",
	)

	enrichCmd = flag.NewFlagSet("enrich", flag.ExitOnError)

	subcommands = map[string]*flag.FlagSet{
		githubCmd.Name():   githubCmd,
		circleCICmd.Name(): circleCICmd,
		enrichCmd.Name():   enrichCmd,
	}
)

func main() {
	// Display list of commands and exit
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" {
		printAvailableCommands()
		os.Exit(0)
	}

	// Parse and validate subcommand
	// The first argument on the command line is the command
	cmd := subcommands[os.Args[1]]
	if cmd == nil {
		log.Fatalf("Unknown subcommand '%s', see help for more details.", os.Args[1])
	}

	switch cmd.Name() {

	case githubCmd.Name():
		cmdGitHub(cmd)

	case circleCICmd.Name():
		cmdCircleCI(cmd)

	case enrichCmd.Name():
		setupCommonFlags()
		cmd.Parse(os.Args[2:])
		validateCommonParams()
		initLogging()

		db := database.GetDB(neo4jURI, neo4jUser, neo4jPassword)
		en := enrich.GetEnricher(db, organization)
		en.Enrich()

	default:
		log.Fatalf("Unknown subcommand '%s', see help for more details.", os.Args[1])
	}
}

// Print list of commands
func printAvailableCommands() {
	fmt.Printf("Usage: %s [SUBCOMMAND] [OPTIONS]...\nAvailable subcommands:\n", os.Args[0])
	keys := make([]string, 0, len(subcommands))
	for k := range subcommands {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		fmt.Printf("\t%s\n", name)
	}
}

// Initialize logging
func initLogging() {
	logLevel := log.InfoLevel
	if debug {
		logLevel = log.DebugLevel
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(logLevel)
}

// Set up common flags used by all commands.
func setupCommonFlags() {
	for _, fs := range subcommands {
		fs.StringVar(&organization, "organization", "", "The target GitHub organization slug.")
		fs.StringVar(&neo4jURI, "neo4j-uri", "neo4j://localhost:7687", "The Neo4j URI.")
		fs.StringVar(&neo4jUser, "neo4j-user", "neo4j", "The Neo4j user.")
		fs.StringVar(&neo4jPassword, "neo4j-password", "", "The Neo4j password.")
		fs.BoolVar(&debug, "debug", false, "Enable debug logging.")
	}
}

// Validate commong flags used by all commands.
func validateCommonParams() {
	requiredFlags := map[string]string{
		organization:  "-organization",
		neo4jPassword: "-neo4j-password",
	}

	for k, v := range requiredFlags {
		if k == "" {
			log.Fatalf("The %s flag is required for all commands.", v)
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
	}
	defaultNames := []string{
		"organizations",
		"teams",
		"users",
		"repos",
		"teamrepos",
		"teammembers",
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

// Returns true if slice s contains element e, false otherwise.
func sliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Returns lowercased slice of strings.
func sliceLower(s []string) []string {
	r := []string{}
	for _, a := range s {
		r = append(r, strings.ToLower(a))
	}
	return r
}

// Returns slice s with all occurences of e removed.
func sliceRemove(s []string, e string) []string {
	var hit bool
	for {
		hit = false
		for i, a := range s {
			if a == e {
				s = append(s[:i], s[i+1:]...)
				hit = true
				break
			}
		}
		if !hit {
			break
		}
	}
	return s
}

// Returns slice s with duplicates removed.
func sliceDeduplicate(s []string) []string {
	keys := make(map[string]bool)
	r := []string{}
	for _, e := range s {
		if _, seen := keys[e]; !seen {
			keys[e] = true
			r = append(r, e)
		}
	}
	return r
}
