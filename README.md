# ingest

> Version Control Security

Ingests GitHub organization data into Neo4j.

This will allow us to identify lateral movement and privilege escalation opportunities in GitHub, _à la_ BloodHound or Cartography.

This is still early stages.

# Kickoff Document

[Confluence](https://ovotech.atlassian.net/wiki/spaces/SJYMC/pages/3277586461/VCS+Provider+Security)

## Tests

```
$ export NEO4J_PASSWORD=password
$ docker run -p7474:7474 -p7687:7687 --name ingest -e NEO4J_AUTH=neo4j/$NEO4J_PASSWORD neo4j:latest
$ go test -v ./...
```

## Run

```
$ export NEO4J_PASSWORD=password
$ docker run -p7474:7474 -p7687:7687 --name ingest -e NEO4J_AUTH=neo4j/$NEO4J_PASSWORD neo4j:latest
$ go run cmd/main.go
```
