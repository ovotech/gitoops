module "repos" {
  count = length(var.repos)

  source = "./modules/repo"

  org        = var.org_name
  name       = var.repos[count.index].name
  files_path = "${var.repos_path}/${var.repos[count.index].name}"
  branch     = var.branch_name
  circleci   = var.repos[count.index].circleci

  providers = {
    github = github.github
	circleci = circleciprojects
  }
}

resource "github_team_repository" "some_team_repo" {
  count = length(var.team_repos)
  team_id = github_team.team[
    index(github_team.team.*.name, var.memberships[count.index].team)
  ].id
  repository = module.repos[
    index(module.repos.*.repo_name, var.team_repos[count.index].repo)
  ].repo_id
  permission = "pull"
}
