package github

import (
	"fmt"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	files := []string{
		"issues_event.json",
		"label_event.json",
		"milestone_event.json",
		"push_event.json",
	}

	for _, path := range files {
		t.Run(fmt.Sprintf("with event %s", path), func(t *testing.T) {
			t.Setenv("GITHUB_ACTIONS", "true")
			t.Setenv("GITHUB_HEAD_REF", "")
			t.Setenv("GITHUB_ACTOR", "tjamet")
			t.Setenv("GITHUB_ACTION", "run2")
			t.Setenv("GITHUB_REF", "refs/heads/master")
			t.Setenv("GITHUB_SHA", "d74fd518cf0410699c6b748924727686c1606d00")
			t.Setenv("GITHUB_EVENT_PATH", "/home/runner/work/_temp/_github_workflow/event.json")
			t.Setenv("GITHUB_BASE_REF", "")
			t.Setenv("GITHUB_REPOSITORY", "tjamet/actions-playground")
			t.Setenv("GITHUB_EVENT_NAME", "push")
			t.Setenv("GITHUB_WORKFLOW", "CI")
			t.Setenv("GITHUB_WORKSPACE", "/home/runner/work/actions-playground/actions-playground")
			t.Setenv("GITHUB_EVENT_PATH", path)
			act, err := ParseActionEnv()
			require.NoError(t, err)

			snaps.MatchSnapshot(t, path, act)
			// assert.Equal(t, deserializeAnonymous(bytes.NewReader(data)), deserializeAnonymous(b))
		})
	}
}
