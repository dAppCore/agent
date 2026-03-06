package forgejo

import (
	"testing"

	forgejosdk "codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v2"
	"github.com/stretchr/testify/assert"
)

func TestMapPRState_Good_Open(t *testing.T) {
	pr := &forgejosdk.PullRequest{State: forgejosdk.StateOpen, HasMerged: false}
	assert.Equal(t, "OPEN", mapPRState(pr))
}

func TestMapPRState_Good_Merged(t *testing.T) {
	pr := &forgejosdk.PullRequest{State: forgejosdk.StateClosed, HasMerged: true}
	assert.Equal(t, "MERGED", mapPRState(pr))
}

func TestMapPRState_Good_Closed(t *testing.T) {
	pr := &forgejosdk.PullRequest{State: forgejosdk.StateClosed, HasMerged: false}
	assert.Equal(t, "CLOSED", mapPRState(pr))
}

func TestMapPRState_Good_UnknownState(t *testing.T) {
	// Any unknown state should default to CLOSED.
	pr := &forgejosdk.PullRequest{State: "weird", HasMerged: false}
	assert.Equal(t, "CLOSED", mapPRState(pr))
}

func TestMapMergeable_Good_Mergeable(t *testing.T) {
	pr := &forgejosdk.PullRequest{Mergeable: true, HasMerged: false}
	assert.Equal(t, "MERGEABLE", mapMergeable(pr))
}

func TestMapMergeable_Good_Conflicting(t *testing.T) {
	pr := &forgejosdk.PullRequest{Mergeable: false, HasMerged: false}
	assert.Equal(t, "CONFLICTING", mapMergeable(pr))
}

func TestMapMergeable_Good_Merged(t *testing.T) {
	pr := &forgejosdk.PullRequest{HasMerged: true}
	assert.Equal(t, "UNKNOWN", mapMergeable(pr))
}

func TestMapCombinedStatus_Good_Success(t *testing.T) {
	cs := &forgejosdk.CombinedStatus{
		State:      forgejosdk.StatusSuccess,
		TotalCount: 1,
	}
	assert.Equal(t, "SUCCESS", mapCombinedStatus(cs))
}

func TestMapCombinedStatus_Good_Failure(t *testing.T) {
	cs := &forgejosdk.CombinedStatus{
		State:      forgejosdk.StatusFailure,
		TotalCount: 1,
	}
	assert.Equal(t, "FAILURE", mapCombinedStatus(cs))
}

func TestMapCombinedStatus_Good_Error(t *testing.T) {
	cs := &forgejosdk.CombinedStatus{
		State:      forgejosdk.StatusError,
		TotalCount: 1,
	}
	assert.Equal(t, "FAILURE", mapCombinedStatus(cs))
}

func TestMapCombinedStatus_Good_Pending(t *testing.T) {
	cs := &forgejosdk.CombinedStatus{
		State:      forgejosdk.StatusPending,
		TotalCount: 1,
	}
	assert.Equal(t, "PENDING", mapCombinedStatus(cs))
}

func TestMapCombinedStatus_Good_Nil(t *testing.T) {
	assert.Equal(t, "PENDING", mapCombinedStatus(nil))
}

func TestMapCombinedStatus_Good_ZeroCount(t *testing.T) {
	cs := &forgejosdk.CombinedStatus{
		State:      forgejosdk.StatusSuccess,
		TotalCount: 0,
	}
	assert.Equal(t, "PENDING", mapCombinedStatus(cs))
}

func TestParseEpicChildren_Good_Mixed(t *testing.T) {
	body := "## Sprint\n- [x] #1\n- [ ] #2\n- [x] #3\n- [ ] #4\nSome text\n"
	unchecked, checked := parseEpicChildren(body)
	assert.Equal(t, []int{2, 4}, unchecked)
	assert.Equal(t, []int{1, 3}, checked)
}

func TestParseEpicChildren_Good_NoCheckboxes(t *testing.T) {
	body := "This is just a normal issue with no checkboxes."
	unchecked, checked := parseEpicChildren(body)
	assert.Nil(t, unchecked)
	assert.Nil(t, checked)
}

func TestParseEpicChildren_Good_AllChecked(t *testing.T) {
	body := "- [x] #10\n- [x] #20\n"
	unchecked, checked := parseEpicChildren(body)
	assert.Nil(t, unchecked)
	assert.Equal(t, []int{10, 20}, checked)
}

func TestParseEpicChildren_Good_AllUnchecked(t *testing.T) {
	body := "- [ ] #5\n- [ ] #6\n"
	unchecked, checked := parseEpicChildren(body)
	assert.Equal(t, []int{5, 6}, unchecked)
	assert.Nil(t, checked)
}

func TestFindLinkedPR_Good(t *testing.T) {
	prs := []*forgejosdk.PullRequest{
		{Index: 10, Body: "Fixes #5"},
		{Index: 11, Body: "Resolves #7"},
		{Index: 12, Body: "Nothing here"},
	}

	pr := findLinkedPR(prs, 7)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(11), pr.Index)
}

func TestFindLinkedPR_Good_NotFound(t *testing.T) {
	prs := []*forgejosdk.PullRequest{
		{Index: 10, Body: "Fixes #5"},
	}
	pr := findLinkedPR(prs, 99)
	assert.Nil(t, pr)
}

func TestFindLinkedPR_Good_Nil(t *testing.T) {
	pr := findLinkedPR(nil, 1)
	assert.Nil(t, pr)
}

func TestBuildSignal_Good(t *testing.T) {
	pr := &forgejosdk.PullRequest{
		Index:     42,
		State:     forgejosdk.StateOpen,
		Mergeable: true,
		Head:      &forgejosdk.PRBranchInfo{Sha: "deadbeef"},
	}

	sig := buildSignal("org", "repo", 10, 5, pr, "SUCCESS")

	assert.Equal(t, 10, sig.EpicNumber)
	assert.Equal(t, 5, sig.ChildNumber)
	assert.Equal(t, 42, sig.PRNumber)
	assert.Equal(t, "org", sig.RepoOwner)
	assert.Equal(t, "repo", sig.RepoName)
	assert.Equal(t, "OPEN", sig.PRState)
	assert.Equal(t, "MERGEABLE", sig.Mergeable)
	assert.Equal(t, "SUCCESS", sig.CheckStatus)
	assert.Equal(t, "deadbeef", sig.LastCommitSHA)
	assert.False(t, sig.IsDraft)
}

func TestBuildSignal_Good_NilHead(t *testing.T) {
	pr := &forgejosdk.PullRequest{
		Index:     1,
		State:     forgejosdk.StateClosed,
		HasMerged: true,
	}

	sig := buildSignal("org", "repo", 1, 2, pr, "PENDING")
	assert.Equal(t, "", sig.LastCommitSHA)
	assert.Equal(t, "MERGED", sig.PRState)
}

func TestSplitRepo_Good(t *testing.T) {
	tests := []struct {
		input string
		owner string
		repo  string
		err   bool
	}{
		{"host-uk/core", "host-uk", "core", false},
		{"a/b", "a", "b", false},
		{"org/repo-name", "org", "repo-name", false},
		{"invalid", "", "", true},
		{"", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := splitRepo(tt.input)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.owner, owner)
				assert.Equal(t, tt.repo, repo)
			}
		})
	}
}
