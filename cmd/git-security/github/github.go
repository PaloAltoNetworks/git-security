package gh

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type GitHub interface {
	CreateBranchProtectionRule(repoID, pattern string) error
	GetOrganizations() ([]*Organization, error)
	GetRepos(orgName string) ([]*Repository, error)
	UpdateBranchProtectionRule(branchProtectionRuleID, field string, value interface{}) error
}

type GitHubImpl struct {
	ctx        context.Context
	restClient *github.Client
	gqlClient  *githubv4.Client
	githubHost string
}

type customTransport struct {
	rt http.RoundTripper
}

func (ct *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return ct.rt.RoundTrip(req)
}

func New(ctx context.Context, host, pat, caCertPath string) (GitHub, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
		client := &http.Client{Transport: tr}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
	}
	tc := oauth2.NewClient(ctx, ts)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(tc.Transport)
	if err != nil {
		return nil, err
	}

	restClient := github.NewClient(rateLimiter).WithAuthToken(pat)
	gqlClient := githubv4.NewClient(tc)
	if !strings.Contains(host, "github.com") {
		u := fmt.Sprintf("https://%s", host)
		restClient, err = restClient.WithEnterpriseURLs(u, u)
		if err != nil {
			return nil, err
		}
		gqlClient = githubv4.NewEnterpriseClient(fmt.Sprintf("%s/api/graphql", u), tc)
	}

	return &GitHubImpl{
		ctx:        ctx,
		restClient: restClient,
		gqlClient:  gqlClient,
		githubHost: host,
	}, nil
}
