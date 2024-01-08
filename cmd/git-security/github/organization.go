package gh

import (
	"github.com/google/go-github/v57/github"
)

type Organization struct {
	Login string
}

func (ghi *GitHubImpl) GetOrganizations() ([]*Organization, error) {
	orgs := make([]*Organization, 0)
	opts := &github.OrganizationsListOptions{}
	for {
		results, _, err := ghi.restClient.Organizations.ListAll(ghi.ctx, opts)
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			break
		}
		for _, org := range results {
			orgs = append(orgs, &Organization{
				Login: org.GetLogin(),
			})
		}
		opts.Since = *results[len(results)-1].ID
	}
	return orgs, nil
}
