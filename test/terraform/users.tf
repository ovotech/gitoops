// not part of the test/demo infra. these are org owners, if needed. 
resource "github_membership" "owner" {
  count    = length(var.owners)
  username = var.owners[count.index]
  role     = "admin"
}

resource "github_membership" "member" {
  count    = length(var.members)
  username = var.members[count.index]
  role     = "member"
}

resource "github_team" "team" {
  count   = length(var.teams)
  name    = var.teams[count.index]
  privacy = "closed"
}

resource "github_team_membership" "membership" {
  count = length(var.memberships)

  // we get the team id by finding the index of the team in github_team.team
  team_id = github_team.team[
    index(github_team.team.*.name, var.memberships[count.index].team)
  ].id
  username = var.memberships[count.index].member
  role     = "member"
}
