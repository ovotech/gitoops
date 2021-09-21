package e2e

import "testing"

func TestUserRelationships(t *testing.T) {
	var testCases = []testCase{
		{
			a: node{
				label: "User",
				property: property{
					name:  "login",
					value: "alice-fwaa",
				},
			},
			r: relationship{
				label: "IS_MEMBER_OF",
			},
			b: node{
				label: "Team",
				property: property{
					name:  "name",
					value: "admin",
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(tc, t)
	}
}
