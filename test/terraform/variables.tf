variable "org_name" {
  default = "failwhales"
}

variable "org_short" {
  default = "fw"
}

variable "owners" {
  type    = list(string)
  default = ["serain"]
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