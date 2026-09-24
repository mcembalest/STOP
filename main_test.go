package main

import "testing"

func TestSTOPHasOneGoalAndRoles(t *testing.T) {
	good := "# GOAL\nA goal.\n# ROLE One\nFirst job.\n# ROLE Two\nSecond job.\n"
	file, err := readSTOP(good)
	if err != nil || file.goal != "A goal." || len(file.roles) != 2 {
		t.Fatalf("got %+v, %v", file, err)
	}
	for _, bad := range []string{
		"# GOAL\nA.\n# GOAL\nB.\n# ROLE One\nJob.\n",
		"# GOAL\nA.\n",
		"# ROLE One\nJob.\n# GOAL\nA.\n",
	} {
		if _, err := readSTOP(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
