variable "org_name" {
  default = "failwhales"
}

variable "org_short" {
  default = "fw"
}

variable "owners" {
  type    = list(string)
  default = ["serain", "congon4tor"]
}

variable "members" {
  type = list(string)
  default = [
    "alice",
    "bob",
    "charlotte",
    "daniel",
    "ellie"
  ]
}

variable "teams" {
  type = list(string)
  default = [
    "admin",
    "infra",
    "payments",
    "data",
    "frontend",
  ]
}

variable "memberships" {
  type = list(object({
    team   = string
    member = string
  }))
  default = [
    {
      team   = "admin"
      member = "alice"
    },
    {
      team   = "infra"
      member = "bob"
    },
    {
      team   = "payments"
      member = "charlotte"
    },
    {
      team   = "data"
      member = "daniel"
    },
    {
      team   = "frontend"
      member = "ellie"
    }
  ]
}

variable "repos_path" {
  type    = string
  default = "./data/repos"
}

variable "repos" {
  type = list(object({
    name     = string
    circleci = bool
  }))
  default = [
    {
      name     = "aws-infra"
      circleci = true
    },
    {
      name     = "console-spa"
      circleci = false
    },
	{
		name = "aws-infra-copy"
		circleci = true
	},
	{
		name = "aws-infra-clone"
		circleci = true
	}
  ]
}

variable "team_repos" {
  type = list(object({
    team       = string
    repo       = string
    permission = string
  }))
  default = [
    {
      team       = "infra"
      repo       = "aws-infra"
      permission = "admin"
    },
    {
      team       = "frontend"
      repo       = "console-spa"
      permission = "admin"
    }
  ]
}

variable "branch_name" {
  description = "Name of branch we use to trigger CI on our repos"
  default     = "trigger"
}