package service

import (
	"testing"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	gh "github.com/PaloAltoNetworks/git-security/cmd/git-security/github"
	"github.com/stretchr/testify/assert"
)

func TestProceedWithRightCondition(t *testing.T) {
	repo := &gh.Repository{
		GqlRepository: &gh.GqlRepository{
			ID:            "foobar",
			NameWithOwner: "org/repo123",
		},
		RepoOwner: "foobar",
	}
	assert.True(t, proceedWithRightCondition(repo, config.Automation{
		Pattern: "org/repo*",
		Owner:   "",
		Exclude: "",
	}))
	assert.False(t, proceedWithRightCondition(repo, config.Automation{
		Pattern: "org/repo*",
		Owner:   "f",
		Exclude: "",
	}))
	assert.True(t, proceedWithRightCondition(repo, config.Automation{
		Pattern: "org/repo*",
		Owner:   ",fooba*",
		Exclude: "",
	}))
	assert.True(t, proceedWithRightCondition(repo, config.Automation{
		Pattern: "org/repo*",
		Owner:   ",123,fooba*",
		Exclude: "",
	}))
	assert.False(t, proceedWithRightCondition(repo, config.Automation{
		Pattern: "org/repo*",
		Owner:   ",123,fooba*",
		Exclude: "org/repo123",
	}))
}
