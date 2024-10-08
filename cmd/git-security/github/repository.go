package gh

import (
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/PaloAltoNetworks/git-security/cmd/git-security/config"
	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	preReceiveHookEnabled  = "enabled"
	preReceiveHookDisabled = "disabled"
)

type Repository struct {
	*GqlRepository           `bson:"inline"`
	Customs                  map[string]interface{} `bson:"customs,omitempty" json:"customs,omitempty" diff:"Customs"`
	GitHubHost               string                 `bson:"github_host,omitempty" json:"github_host,omitempty"`
	Score                    *int                   `bson:"score,omitempty" json:"score,omitempty"`
	ScoreColor               *string                `bson:"score_color,omitempty" json:"score_color,omitempty"`
	FetchedAt                time.Time              `bson:"fetched_at,omitempty" json:"fetched_at,omitempty"`
	CustomRunAt              time.Time              `bson:"custom_run_at,omitempty" json:"custom_run_at,omitempty"`
	LastCommittedAt          time.Time              `bson:"last_committed_at" json:"last_committed_at"`
	RepoOwnerID              primitive.ObjectID     `bson:"repo_owner_id,omitempty" json:"repo_owner_id,omitempty"`
	RepoOwner                string                 `bson:"repo_owner,omitempty" json:"repo_owner,omitempty"`
	RepoOwnerContact         string                 `bson:"repo_owner_contact,omitempty" json:"repo_owner_contact,omitempty"`
	AutomationsCount         int                    `bson:"automations_count,omitempty" json:"automations_count"`
	BypassPullRequestUsers   []string               `bson:"bypass_pull_request_users,omitempty" json:"bypass_pull_request_users,omitempty"`
	BypassPullRequestUserIDs []string               `bson:"bypass_pull_request_user_ids,omitempty" json:"bypass_pull_request_user_ids,omitempty"`
	PushAllowanceUsers       []string               `bson:"push_allowance_users,omitempty" json:"push_allowance_users,omitempty"`
	PushAllowanceUserIDs     []string               `bson:"push_allowance_user_ids,omitempty" json:"push_allowance_user_ids,omitempty"`
}

type GqlRepository struct {
	ID            string `bson:"id" json:"id"`
	Name          string `bson:"name" json:"name"`
	NameWithOwner string `bson:"full_name" json:"full_name"`
	Owner         struct {
		Login string `bson:"login" json:"login"`
	} `bson:"owner" json:"owner"`
	DefaultBranchRef struct {
		Name                 string               `bson:"name" json:"name" diff:"DefaultBranch"`
		BranchProtectionRule BranchProtectionRule `bson:"branch_protection_rule" json:"branch_protection_rule"`
		Target               Target               `bson:"target" json:"target"`
	} `bson:"default_branch" json:"default_branch"`
	PrimaryLanguage struct {
		Name string `bson:"name" json:"name"`
	} `bson:"primary_language" json:"primary_language" diff:"Language"`
	PullRequests        PullRequests `bson:"pull_requests" json:"pull_requests"`
	Refs                Refs         `bson:"refs" json:"refs" graphql:"refs(first: 0, refPrefix: \"refs/heads/\")"`
	IsArchived          bool         `bson:"is_archived" json:"is_archived"`
	IsDisabled          bool         `bson:"is_disabled" json:"is_disabled"`
	IsEmpty             bool         `bson:"is_empty" json:"is_empty"`
	IsLocked            bool         `bson:"is_locked" json:"is_locked"`
	IsPrivate           bool         `bson:"is_private" json:"is_private"`
	DeleteBranchOnMerge bool         `bson:"delete_branch_on_merge" json:"delete_branch_on_merge"`
	MergeCommitAllowed  bool         `bson:"merge_commit_allowed" json:"merge_commit_allowed"`
	RebaseMergeAllowed  bool         `bson:"rebase_merge_allowed" json:"rebase_merge_allowed"`
	SquashMergeAllowed  bool         `bson:"squash_merge_allowed" json:"squash_merge_allowed"`
	DiskUsage           int          `bson:"disk_usage" json:"disk_usage"`
	CreatedAt           time.Time    `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time    `bson:"updated_at" json:"updated_at"`
}

type Refs struct {
	TotalCount int `bson:"total_count" json:"total_count" diff:"RefsTotalCount"`
}

type PullRequests struct {
	TotalCount int `bson:"total_count" json:"total_count" diff:"PRTotalCount"`
}

type BranchProtectionRule struct {
	ID                           string                      `bson:"id" json:"id" diff:"BranchProtectionRuleID"`
	Pattern                      string                      `bson:"pattern" json:"pattern"`
	AllowsForcePushes            bool                        `bson:"allows_force_pushes" json:"allows_force_pushes"`
	AllowsDeletions              bool                        `bson:"allows_deletion" json:"allows_deletion"`
	BypassPullRequestAllowances  BypassPullRequestAllowances `bson:"-" json:"bypass_pull_request_allowances" graphql:"bypassPullRequestAllowances(first: 100)"`
	DismissesStaleReviews        bool                        `bson:"dismisses_stale_reviews" json:"dismisses_stale_reviews"`
	IsAdminEnforced              bool                        `bson:"is_admin_enforced" json:"is_admin_enforced"`
	PushAllowances               PushAllowances              `bson:"-" json:"push_allowances" graphql:"pushAllowances(first: 100)"`
	RequireLastPushApproval      bool                        `bson:"require_last_push_approval" json:"require_last_push_approval"`
	RequiredApprovingReviewCount int                         `bson:"required_approving_review_count" json:"required_approving_review_count"`
	RequiredStatusChecks         []struct {
		Context string `bson:"context" json:"context"`
	} `bson:"required_status_checks" json:"required_status_checks"`
	RequiresApprovingReviews       bool `bson:"requires_approving_reviews" json:"requires_approving_reviews"`
	RequiresCodeOwnerReviews       bool `bson:"requires_code_owner_reviews" json:"requires_code_owner_reviews"`
	RequiresCommitSignatures       bool `bson:"requires_commit_signatures" json:"requires_commit_signatures"`
	RequiresConversationResolution bool `bson:"requires_conversation_resolution" json:"requires_conversation_resolution"`
	RequiresLinearHistory          bool `bson:"requires_linear_history" json:"requires_linear_history"`
	RequiresStatusChecks           bool `bson:"requires_status_checks" json:"requires_status_checks"`
	RequiresStrictStatusChecks     bool `bson:"requires_strict_status_checks" json:"requires_strict_status_checks"`
	RestrictsPushes                bool `bson:"retricts_pushes" json:"retricts_pushes"`
	RestrictsReviewDismissals      bool `bson:"retricts_review_dismissals" json:"retricts_review_dismissals"`
}

type BypassPullRequestAllowances struct {
	Nodes []BypassPullRequestAllowance
}

type PushAllowances struct {
	Nodes []PushAllowance
}

type BypassPullRequestAllowance struct {
	Actor Actor `bson:"actor" json:"actor"`
}

type PushAllowance struct {
	Actor Actor `bson:"actor" json:"actor"`
}

type Actor struct {
	User User `graphql:"... on User"`
	Team Team `graphql:"... on Team"`
}

type User struct {
	ID    string `bson:"id" json:"id"`
	Login string `bson:"login" json:"login"`
}

type Team struct {
	ID   string `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`
}

type Target struct {
	Commit CommitFragment `bson:"commit" json:"commit" graphql:"... on Commit"`
}

type CommitFragment struct {
	History History `bson:"history" json:"history" graphql:"history(first: 10)"`
}

type History struct {
	Nodes      []Commit `bson:"-" json:"-"`
	TotalCount int      `bson:"total_count" json:"total_count" diff:"HistoryTotalCount"`
}

type Commit struct {
	Committer     GitActor  `bson:"-" json:"-"`
	CommittedDate time.Time `bson:"-" json:"-"`
}

type GitActor struct {
	Name string `bson:"-" json:"-"`
}

func (ghi *GitHubImpl) NewRepository(node GqlRepository) *Repository {
	r := &Repository{
		GqlRepository: &node,
		GitHubHost:    ghi.githubHost,
	}

	for _, commit := range r.DefaultBranchRef.Target.Commit.History.Nodes {
		if _, ok := ghi.ignoredCommitters[commit.Committer.Name]; !ok {
			r.LastCommittedAt = commit.CommittedDate
			break
		}
	}
	return r
}

func (ghi *GitHubImpl) GetRepos(orgName string) ([]*Repository, error) {
	repos := make([]*Repository, 0)

	var q struct {
		Organization struct {
			Repositories struct {
				Nodes    []GqlRepository
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"repositories(first: 100, after: $cursor)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":  githubv4.String(orgName),
		"cursor": (*githubv4.String)(nil),
	}

	for {
		if err := ghi.gqlClient.Query(ghi.ctx, &q, variables); err != nil {
			return nil, err
		}
		for _, node := range q.Organization.Repositories.Nodes {
			r := ghi.NewRepository(node)
			r.FetchedAt = time.Now()

			// sync BypassPullRequestUsers and PushAllowanceUsers
			r.BypassPullRequestUsers = []string{}
			r.BypassPullRequestUserIDs = []string{}
			r.PushAllowanceUsers = []string{}
			r.PushAllowanceUserIDs = []string{}
			if r.DefaultBranchRef.BranchProtectionRule.Pattern != "" {
				for _, n := range r.DefaultBranchRef.BranchProtectionRule.BypassPullRequestAllowances.Nodes {
					r.BypassPullRequestUsers = append(r.BypassPullRequestUsers, n.Actor.User.Login)
					r.BypassPullRequestUserIDs = append(r.BypassPullRequestUserIDs, n.Actor.User.ID)
				}
				sort.Strings(r.BypassPullRequestUsers)
				for _, n := range r.DefaultBranchRef.BranchProtectionRule.PushAllowances.Nodes {
					r.PushAllowanceUsers = append(r.PushAllowanceUsers, n.Actor.User.Login)
					r.PushAllowanceUserIDs = append(r.PushAllowanceUserIDs, n.Actor.User.ID)
				}
				sort.Strings(r.PushAllowanceUsers)
			}

			repos = append(repos, r)
		}
		if !q.Organization.Repositories.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(q.Organization.Repositories.PageInfo.EndCursor)
	}

	return repos, nil
}

func (ghi *GitHubImpl) GetRepo(orgName, repoName string) (*Repository, error) {
	var q struct {
		Repository GqlRepository `graphql:"repository(owner: $org, name: $repo)"`
	}

	variables := map[string]interface{}{
		"org":  githubv4.String(orgName),
		"repo": githubv4.String(repoName),
	}

	if err := ghi.gqlClient.Query(ghi.ctx, &q, variables); err != nil {
		return nil, err
	}

	r := ghi.NewRepository(q.Repository)
	r.FetchedAt = time.Now()

	return r, nil
}

func (ghi *GitHubImpl) CreateBranchProtectionRule(repoID, pattern string) error {
	var m struct {
		CreateBranchProtectionRule struct {
			BranchProtectionRule struct {
				Repository struct {
					Name string
				}
			}
		} `graphql:"createBranchProtectionRule(input: $input)"`
	}

	input := githubv4.CreateBranchProtectionRuleInput{
		RepositoryID: repoID,
		Pattern:      githubv4.String(pattern),
	}

	return ghi.gqlClient.Mutate(ghi.ctx, &m, input, nil)
}

func (ghi *GitHubImpl) UpdateBranchProtectionRule(branchProtectionRuleID, field string, value interface{}) error {
	var m struct {
		UpdateBranchProtectionRule struct {
			BranchProtectionRule struct {
				Repository struct {
					Name string
				}
			}
		} `graphql:"updateBranchProtectionRule(input: $input)"`
	}

	input := githubv4.UpdateBranchProtectionRuleInput{
		BranchProtectionRuleID: branchProtectionRuleID,
	}
	switch field {
	case "RequiresApprovingReviews":
		if v, ok := value.(bool); ok {
			input.RequiresApprovingReviews = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "RequiredApprovingReviewCount":
		if v, ok := value.(int); ok {
			input.RequiredApprovingReviewCount = githubv4.NewInt(githubv4.Int(v))
		}
	case "DismissesStaleReviews":
		if v, ok := value.(bool); ok {
			input.DismissesStaleReviews = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "RequiresCodeOwnerReviews":
		if v, ok := value.(bool); ok {
			input.RequiresCodeOwnerReviews = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "RequiresStatusChecks":
		if v, ok := value.(bool); ok {
			input.RequiresStatusChecks = githubv4.NewBoolean(githubv4.Boolean(v))
		}

	case "RequiresStrictStatusChecks":
		if v, ok := value.(bool); ok {
			input.RequiresStrictStatusChecks = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "RequiresConversationResolution":
		if v, ok := value.(bool); ok {
			input.RequiresConversationResolution = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "RequiresCommitSignatures":
		if v, ok := value.(bool); ok {
			input.RequiresCommitSignatures = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "IsAdminEnforced":
		if v, ok := value.(bool); ok {
			input.IsAdminEnforced = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "AllowsForcePushes":
		if v, ok := value.(bool); ok {
			input.AllowsForcePushes = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	case "AllowsDeletions":
		if v, ok := value.(bool); ok {
			input.AllowsDeletions = githubv4.NewBoolean(githubv4.Boolean(v))
		}
	}

	return ghi.gqlClient.Mutate(ghi.ctx, &m, input, nil)
}

func (repo *Repository) UpdateRepoScoreAndColor(gs *config.GlobalSettings) error {
	b, err := json.Marshal(*repo)
	if err != nil {
		slog.Error("error in json.Marshal()", slog.String("error", err.Error()))
		return err
	}

	score := 0
	for _, weight := range gs.ScoreWeights {
		hit := false
		fieldValue := gjson.GetBytes(b, weight.Field)
		if fieldValue.Exists() {
			switch weight.Comparator {
			case "==":
				if fieldValue.Value() != nil {
					switch v := fieldValue.Value().(type) {
					case bool:
						hit = v == cast.ToBool(weight.Arg)
					case string:
						hit = v == weight.Arg
					case float64:
						hit = v == cast.ToFloat64(weight.Arg)
					}
				} else {
					hit = weight.Arg == ""
				}
			case "!=":
				if fieldValue.Value() != nil {
					switch v := fieldValue.Value().(type) {
					case bool:
						hit = v != cast.ToBool(weight.Arg)
					case string:
						hit = v != weight.Arg
					case float64:
						hit = v != cast.ToFloat64(weight.Arg)
					}
				} else {

					hit = weight.Arg == ""
				}
			case "<":
				if fieldValue.Value() != nil {
					switch v := fieldValue.Value().(type) {
					case string:
						hit = v < weight.Arg
					case float64:
						hit = v < cast.ToFloat64(weight.Arg)
					}
				}
			case "<=":
				if fieldValue.Value() != nil {
					switch v := fieldValue.Value().(type) {
					case string:
						hit = v <= weight.Arg
					case float64:
						hit = v <= cast.ToFloat64(weight.Arg)
					}
				}
			case ">":
				if fieldValue.Value() != nil {
					switch v := fieldValue.Value().(type) {
					case string:
						hit = v > weight.Arg
					case float64:
						hit = v > cast.ToFloat64(weight.Arg)
					}
				}
			case ">=":
				if fieldValue.Value() != nil {
					switch v := fieldValue.Value().(type) {
					case string:
						hit = v >= weight.Arg
					case float64:
						hit = v >= cast.ToFloat64(weight.Arg)
					}
				}
			}
		} else {
			hit = weight.Arg == ""
		}
		if hit {
			score += weight.Weight
		}
	}
	repo.Score = &score
	for _, sc := range gs.ScoreColors {
		if score >= sc.Range[0] &&
			(score < sc.Range[1] || score == 100 && sc.Range[1] == 100) {
			repo.ScoreColor = &sc.Color
			break
		}
	}
	return nil
}

func (ghi *GitHubImpl) ArchiveRepository(repoID string, archive bool) error {
	var a struct {
		ArchiveRepository struct {
			Repository struct {
				Name string
			}
		} `graphql:"archiveRepository(input: $input)"`
	}
	ainput := githubv4.ArchiveRepositoryInput{
		RepositoryID: repoID,
	}

	var u struct {
		UnarchiveRepository struct {
			Repository struct {
				Name string
			}
		} `graphql:"unarchiveRepository(input: $input)"`
	}
	uinput := githubv4.UnarchiveRepositoryInput{
		RepositoryID: repoID,
	}

	if archive {
		slog.Info("archiving repo", slog.String("repoID", repoID))
		return ghi.gqlClient.Mutate(ghi.ctx, &a, ainput, nil)
	}
	slog.Info("unarchiving repo", slog.String("repoID", repoID))
	return ghi.gqlClient.Mutate(ghi.ctx, &u, uinput, nil)
}

func (ghi *GitHubImpl) UpdatePreceiveHook(orgName string, repoName string, hookName string, enabled bool) error {
	var hook *github.PreReceiveHook
	opts := &github.ListOptions{
		Page: 1,
	}
	for {
		results, _, err := ghi.restClient.Repositories.ListPreReceiveHooks(ghi.ctx, orgName, repoName, opts)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			break
		}
		for _, h := range results {
			if *h.Name == hookName {
				hook = h
				break
			}
		}
		if hook != nil {
			break
		}
		opts.Page += 1
	}
	if hook != nil {
		if enabled {
			hook.Enforcement = &preReceiveHookEnabled
		} else {
			hook.Enforcement = &preReceiveHookDisabled
		}
		_, _, err := ghi.restClient.Repositories.UpdatePreReceiveHook(ghi.ctx, orgName, repoName, *hook.ID, hook)
		if err != nil {
			return err
		}
	}
	return nil
}
