# Examples

This document shows some example queries to get started.

<details>
<summary>Show me all CI/CD environment variables my GitHub user has direct or indirect acces to</summary>
<br>
This query will return paths between your user and potential secrets, via several means:

- CircleCI
  - "All members" contexts
  - team-restricted contexts
  - repository projects
- GitHub Actions
  - repository
  - environment
  - organization

<pre>
MATCH p=(:User{login:"alice"})-[*..5]->(:EnvironmentVariable)
RETURN p
</pre>

</details>

<details>
<summary>I just compromised a GitHub user called superbot. What new repositories did I gain access to?</summary>
<br>
<pre>
MATCH (:User{login:"superbot"})-->(r:Repository)
WHERE NOT EXISTS((:User{login:'serain'})-->(r))
RETURN r.name
</pre>
</details>

<details>
<summary>Show me GitHub Actions secrets without branch protections</summary>
<br>
To find GitHub Actions environment variables that are not in environments (and therefor accessible to anyone who can open a pull request), we can search for direct relationships between a repository and environment variables:

<pre>
MATCH p=(:Repository)-->(:EnvironmentVariable)
RETURN p
</pre>

Environments also needn't enforce any branch protections. We can look for environment variables that can be exfiltrated from any environment through a pull request:

<pre>
MATCH p=(:Repository)-->(e:Environment)-->(:EnvironmentVariable)
WHERE e.protectedBranches = false
RETURN p
</pre>

</details>

<details>
<summary>Show me repositories running production terraform plan on pull requests</summary>
<br>
Production Terraform plans on unreviewed code are [a bad idea](https://alex.kaskaso.li/post/terraform-plan-rce). We attempt to find these by looking at the context values on pull requests' status checks, to get maximum coverage and account for CI/CD systems that may be configured server-side (e.g. AWS CodeBuild).

<pre>
MATCH (r:Repository)-[:HAS_STATUS_CHECK{pullRequest:TRUE}]->(s:StatusCheck)
WHERE s.context =~ "(?=.*(tf|terraform))(?=.*(?<!non)pro?d).*"
RETURN r.name
</pre>

</details>

<details>
<summary>Show me AWS_SECRET_ACCESS_KEY variables my user can access through WRITE access to repositories with CircleCI project environment variables</summary>
<br>
CircleCI doesn't support branch-level protections for secrets. The implication is that if you can open a PR against a repository, you can exfiltrate secrets from the CI/CD context. These could be production secrets.

Note that our query matches repositories that the user can access both directly and indirectly, through team memberships.

This query is broken down to illustrate the relationships GitOops builds up; the query could be written more succinctly.

<pre>
MATCH p=(:User{login:"serain"})-[*..2]->(:Repository)-[:HAS_CI]->(:CircleCIProject)-[:EXPOSES_ENVIRONMENT_VARIABLE]->(v:EnvironmentVariable)
WHERE v.variable =~ ".*AWS.*SECRET.*"
RETURN p
</pre>
</details>

<details>
<summary>Show me repositories that many users have access to with potentially interesting secrets exposed in a pull request's CI/CD context</summary>
<br>
Here we're using the content of CI/CD configuration files to make educated guesses about interesting pipelines. This is less accurate that using our other relationships, but gives us coverage of unsupported CI/CD systems (as long as we pulled the configuration files).

<pre>
MATCH (r:Repository)-[HAS_CI_CONFIGURATION]->(f:File{path: ".circleci/config.yml"})
WHERE any(x IN f.env WHERE x =~ ".*(AUTH|SECRET|TOKEN|PASS|PWD|CRED|KEY|PRD|PROD).*")
OR any(x IN f.tags WHERE x IN ["aws", "gcp", "terraform"])
WITH r
<br>
MATCH (u:User)-[*..2]->(r)-[HAS_STATUS_CHECK{pullRequest:true}]->(s:StatusCheck)
WITH r, COUNT(DISTINCT u) AS userCount
WHERE userCount > 30
RETURN r
</pre>

</details>

<details>
<summary>Show me repositories with potential automated deployments that don't have branch protections</summary>
<br>
<pre>
MATCH (r:Repository)-[HAS_CI_CONFIGURATION]->(f:File)
WHERE any(x IN f.env WHERE x =~ ".*(AUTH|SECRET|TOKEN|PASS|PWD|CRED|KEY|PRD|PROD).*")
OR any(x IN f.tags WHERE x IN ["aws", "gcp", "terraform"])
WITH r
<br>
MATCH (r)
WHERE NOT (r)-[HAS_BRANCH_PROTECTION_RULE]->(:BranchProtectionRule)
RETURN r

</pre>
</details>

<details>
<summary>Do any external contributors have paths to sensitive secrets?</summary>
<br>
<pre>
MATCH (u:User)
WHERE NOT (u)-[:IS_MEMBER_OF]->(:Organization{login:"fakenews"})
WITH u
<br>
MATCH p=(u)-[*..5]->(v:EnvironmentVariable)
WHERE v.variable =~ "._(AUTH|SECRET|TOKEN|PASS|PWD|CRED|KEY|PRD|PROD)._"
RETURN p

</pre>
</details>

<details>
<summary>Show me who's still using Jenkins</summary>
<br>
<pre>
MATCH p=(t:Team)-[:HAS_PERMISSION_ON]->(r:Repository{isArchived:FALSE})-[:HAS_CI_CONFIGURATION_FILE]->(f:File{path:"Jenkinsfile"})
RETURN p
</pre>
</details>
