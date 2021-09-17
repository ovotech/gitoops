# GitOops Test Infrastructure

We manage the [`failwhales`](https://github.com/failwhales) GitHub organization for testing GitOops end-to-end and provide demo material.

The following steps were manual:

- Create the organization
- Add CircleCI from the Marketplace
- Create all users listed in `var.members` (these users are suffixed with `var.org_short`)
- Accept org invitations on behalf of the users

The rest is defined here as Terraform.

There is currently no CI/CD for this. Applying this requires a user token with `admin:org` and `repo` scopes for a `failwhales` user with `OWNER` permissions.

TODO - DON'T MERGE ME UNTIL THIS IS DONE: Secrets for the users are in the Kaluza Security Engineering team's password manager.