package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v59/github"
	"github.com/haapjari/glass-api/api"
	"github.com/haapjari/glass-api/internal/pkg/cfg"
	"github.com/haapjari/glass-api/internal/pkg/logger"
	"github.com/haapjari/glass-api/internal/pkg/utils"
)

// TODO: Inspect Rate Limit Headers -> Workaround.

// RepositorySearchService godoc.
type RepositorySearchService struct {
	log logger.Logger
	// TODO: Expose this username
	gitHubUsername            string
	gitHubPersonalAccessToken string
	gitHubQueryInterval       time.Duration
	httpClient                *http.Client
	gitHubClient              *github.Client
}

type Count struct {
	TotalCount int `json:"total_count"`
}

type RepositoryResponse struct {
	TotalCount int              `json:"total_count"`
	Items      []api.Repository `json:"items"`
}

func NewRepositorySearchService(logger logger.Logger, config *cfg.Config, token string) (*RepositorySearchService, error) {
	interval, err := time.ParseDuration(config.GitHubQueryInterval)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: time.Duration(30) * time.Second,
	}

	return &RepositorySearchService{
		log:                       logger,
		gitHubPersonalAccessToken: token,
		gitHubQueryInterval:       interval,
		httpClient:                httpClient,
		gitHubClient: github.NewClient(httpClient).
			WithAuthToken(token),
		gitHubUsername: config.GitHubUser,
	}, nil
}

func (rss *RepositorySearchService) Populate() {
	// TODO
}

// Search is an abstraction of GitHub Repository Search API.
// Returns slice of repositories, count, status and optionally an error.
// TODO
func (rss *RepositorySearchService) Search(language string, stars string, firstCreationDate string,
	lastCreationDate string, order string) ([]api.Repository, int, int, error) {
	if language == "" || stars == "" {
		return nil, 0, 400, errors.New("language or stars field is empty")
	}

	var (
		queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:%s..%s&order=%s",
			language, stars, firstCreationDate, lastCreationDate, order)
		endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, 500, err
	}

	rss.log.Debugf("Request: %v", endpoint)

	if rss.gitHubPersonalAccessToken != "" {
		req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
	}

	resp, err := rss.httpClient.Do(req)
	if err != nil {
		return nil, 0, 500, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			rss.log.Warnf("unable to close response body: %s", err.Error())
		}
	}()

	rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

	if resp.StatusCode == 200 {
		repositoryResponse := &RepositoryResponse{}

		body, e := io.ReadAll(resp.Body)
		if e != nil {
			return nil, 0, 500, e
		}

		if err = json.Unmarshal(body, &repositoryResponse); err != nil {
			return nil, 0, 500, err
		}

		// Populate Contributor Count.
		// TODO: Create Populator Goroutine, and
		for i := range len(repositoryResponse.Items) {
			owner, name, loopErr := utils.ParseGitHubFullName(repositoryResponse.Items[i].FullName)
			if loopErr != nil {
				return nil, 0, 500, loopErr
			}

			rss.log.Debugf("[Owner: %v] [Name: %v] populating repo ", owner, name)

			contributorCount, loopErr := rss.GetContributorCount(owner, name)
			if loopErr != nil {
				rss.log.Errorf("unable to get contributor count: %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			latestRelease, loopErr := rss.GetLatestRelease(owner, name)
			if loopErr != nil {
				rss.log.Errorf("unable to get latest release: %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			totalReleases, loopErr := rss.GetTotalReleases(owner, name)
			if loopErr != nil {
				rss.log.Errorf("unable to get total releases: %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			openPullsCount, loopErr := rss.GetOpenPullRequests(owner, name)
			if loopErr != nil {
				rss.log.Errorf("unable to get open pull requests: %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			closedPullsCount, loopErr := rss.GetClosedPullRequests(owner, name)
			if loopErr != nil {
				rss.log.Errorf("unable to get closed pull requests: %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			req, err = http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s",
				owner, name), nil)
			if err != nil {
				rss.log.Errorf("unable to construct http request: %s", err.Error())
				return nil, 0, 500, err
			}

			resp, err = rss.httpClient.Do(req)
			if err != nil {
				rss.log.Errorf("unable to execute http request: %s", err.Error())
				return nil, 0, 500, err
			}

			repo := Repository{}

			if resp.StatusCode == 200 {
				if err = json.NewDecoder(resp.Body).Decode(&repo); err != nil {
					if err != nil {
						rss.log.Errorf("unable to decode response %s", err.Error())
						return nil, 0, 500, err
					}
				}
			}

			err = resp.Body.Close()
			if err != nil {
				rss.log.Errorf("unable to close body: %s", err.Error())
				return nil, 0, 500, err
			}

			commitsCount, loopErr := rss.GetCommitsCount(owner, name)
			if loopErr != nil {
				rss.log.Errorf("unable to get commits count: %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			// TODO: There is an error, "GetLinesOfCode" requires access to
			// authentication.
			selfWrittenLOC, libraryLOC, loopErr := rss.GetLinesOfCode(name, repo.GitURL)
			if loopErr != nil {
				rss.log.Errorf("unable to get : %s", loopErr.Error())
				return nil, 0, 500, loopErr
			}

			repositoryResponse.Items[i].ContributorsCount = &contributorCount
			repositoryResponse.Items[i].LatestRelease = &latestRelease
			repositoryResponse.Items[i].TotalReleasesCount = &totalReleases
			repositoryResponse.Items[i].OpenPullsCount = &openPullsCount
			repositoryResponse.Items[i].ClosedPullsCount = &closedPullsCount
			repositoryResponse.Items[i].SubscribersCount = &repo.SubscribersCount
			repositoryResponse.Items[i].CommitsCount = &commitsCount
			repositoryResponse.Items[i].NetworkCount = &repo.NetworkCount
			repositoryResponse.Items[i].WatchersCount = &repo.Watchers
			repositoryResponse.Items[i].SelfWrittenLoc = &selfWrittenLOC
			repositoryResponse.Items[i].LibraryLoc = &libraryLOC
			repositoryResponse.Items[i].SelfWrittenLoc = &selfWrittenLOC
			repositoryResponse.Items[i].LibraryLoc = &libraryLOC
		}

		return repositoryResponse.Items, repositoryResponse.TotalCount, 200, nil
	}

	return nil, 0, 500, nil
}

// GetLinesOfCode returns "Self-Written LOC" and "Library LOC".
func (rss *RepositorySearchService) GetLinesOfCode(name, remote string) (int, int, error) {
	remote = strings.Replace(remote, "git://", "https://", 1)
	baseDir := os.TempDir()
	dir, err := os.MkdirTemp(baseDir, name+"-*")
	if err != nil {
		return -1, -1, err
	}

	repo := NewRepo(remote, dir, name, rss.gitHubUsername,
		rss.gitHubPersonalAccessToken, rss.log)

	rss.log.Debugf("Cloning Repository '%v' into %v.", remote, dir)

	err = repo.Clone()
	if err != nil {
		return -1, -1, err
	}

	selfWrittenLOC, err := repo.SelfWrittenLOC()
	if err != nil {
		rss.log.Errorf("unable to calculate self-written loc: %s", err.Error())

		return -1, -1, err
	}

	libraryLOC, err := repo.CalculateLibraryLOC()
	if err != nil {
		return -1, -1, err
	}

	rss.log.Debugf("Deleting Repository '%v', from %v", remote, dir)

	err = repo.Delete()
	if err != nil {
		return -1, -1, err
	}

	return selfWrittenLOC, libraryLOC, nil
}

// GetCommitsCount godoc
func (rss *RepositorySearchService) GetCommitsCount(owner, repo string) (int, error) {
	page := 1
	count := 0

	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=100&page=%d", owner, repo, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return 0, err
		}

		resp, err := rss.httpClient.Do(req)
		if err != nil {
			return count, err
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == 200 {
			var commits []interface{}
			if err = json.NewDecoder(resp.Body).Decode(&commits); err != nil {
				return count, err
			}

			count += len(commits)

			if linkHeader := resp.Header.Get("Link"); !strings.Contains(linkHeader, `rel="next"`) {
				break
			}

			page++
		} else {
			return count, fmt.Errorf("GitHub API error: %s", resp.Status)
		}
	}

	return count, nil
}

// GetOpenPullRequests godoc
func (rss *RepositorySearchService) GetOpenPullRequests(owner, repo string) (int, error) {

	page := 1
	count := 0

	for {
		req, err := http.NewRequest("GET",
			fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=open&per_page=100&page=%d",
				owner, repo, page), nil)
		if err != nil {
			return 0, err
		}

		resp, e := rss.httpClient.Do(req)
		if e != nil {
			return count, e
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == 200 {
			var pullRequests []interface{}
			if err = json.NewDecoder(resp.Body).Decode(&pullRequests); err != nil {
				return count, err
			}

			count += len(pullRequests)

			if linkHeader := resp.Header.Get("Link"); !strings.Contains(linkHeader, `rel="next"`) {
				break
			} else {
				page = page + 1
			}
		}
	}

	return count, nil
}

// GetClosedPullRequests godoc
func (rss *RepositorySearchService) GetClosedPullRequests(owner, repo string) (int, error) {
	page := 1
	count := 0

	for {
		req, err := http.NewRequest("GET",
			fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=closed&per_page=100&page=%d",
				owner, repo, page), nil)
		if err != nil {
			return 0, err
		}

		resp, e := rss.httpClient.Do(req)
		if e != nil {
			return count, e
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == 200 {
			var pullRequests []interface{}
			if err = json.NewDecoder(resp.Body).Decode(&pullRequests); err != nil {
				return count, err
			}

			count += len(pullRequests)

			if linkHeader := resp.Header.Get("Link"); !strings.Contains(linkHeader, `rel="next"`) {
				break
			} else {
				page = page + 1
			}
		}
	}

	return count, nil
}

// GetTotalReleases godoc
func (rss *RepositorySearchService) GetTotalReleases(owner string, name string) (int, error) {
	opt := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	var all []*github.RepositoryRelease

	for {
		releases, resp, err := rss.gitHubClient.Repositories.ListReleases(context.Background(), owner, name, opt)
		if err != nil {
			return -1, err
		}

		all = append(all, releases...)
		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return len(all), nil
}

// GetLatestRelease godoc
func (rss *RepositorySearchService) GetLatestRelease(owner string, name string) (string, error) {
	release, _, err := rss.gitHubClient.Repositories.GetLatestRelease(context.Background(), owner, name)
	if err != nil {
		return "", err
	}

	return release.GetCreatedAt().String(), nil
}

// GetContributorCount godoc
func (rss *RepositorySearchService) GetContributorCount(owner string, name string) (int, error) {
	opt := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		Anon: "true",
	}

	var all []*github.Contributor

	for {
		contributors, resp, err := rss.gitHubClient.Repositories.ListContributors(context.Background(),
			owner, name, opt)
		if err != nil {
			return -1, err
		}

		all = append(all, contributors...)
		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return len(all), nil
}

// LastCreationDate queries GitHub Search API and returns the last creation date matching the query parameters.
// Returns date as string, API Status Code and error.
func (rss *RepositorySearchService) LastCreationDate(language string, stars string) (string, int, error) {
	if language == "" || stars == "" {
		return "", 400, errors.New("language or stars field is empty")
	}

	var (
		date  = time.Now()
		year  = date.Year()
		month = int(date.Month())
		day   = date.Day()
	)

	startDate, e := time.Parse(time.DateOnly, fmt.Sprintf("%d-%02d-%02d", year, month, day))
	if e != nil {
		return "", 500, e
	}

	var totalCount int

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:>%s&order=asc",
					language, stars, startDate.Format(time.DateOnly))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return "", 500, err
			}

			rss.log.Debugf("Request: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return "", 500, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return "", 500, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return "", 500, err
				}

				totalCount = count.TotalCount

				if totalCount != 0 {
					return startDate.Format(time.DateOnly), 200, nil
				}

				startDate = startDate.AddDate(0, 0, -1)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
}

// FirstCreationDate queries GitHub Search API and returns the first creation date matching the query parameters.
// Returns date as string, API Status Code and error.
func (rss *RepositorySearchService) FirstCreationDate(language string, stars string) (string, int, error) {
	if language == "" || stars == "" {
		return "", 400, errors.New("language or stars field is empty")
	}

	year, err := rss.findFirstYear(language, stars)
	if err != nil {
		return "", 500, err
	}

	month, err := rss.findFirstMonth(language, stars, year)
	if err != nil {
		return "", 500, err
	}

	weekDay, err := rss.findFirstWeek(language, stars, year, month)
	if err != nil {
		return "", 500, err
	}

	day, err := rss.findFirstDay(language, stars, year, month, weekDay)
	if err != nil {
		return "", 500, err
	}

	return fmt.Sprintf("%d-%02d-%02d", year, month, day), 200, nil
}

func (rss *RepositorySearchService) findFirstYear(language string, stars string) (int, error) {
	startDate, e := time.Parse("2006-01-02", "2007-01-01")
	if e != nil {
		return -1, e
	}

	var totalCount int

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount

				if totalCount != 0 {
					return startDate.Year() - 1, nil
				}

				startDate = startDate.AddDate(1, 0, 0)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
}

func (rss *RepositorySearchService) findFirstMonth(language string, stars string, year int) (int, error) {
	startDate, e := time.Parse("2006-01-02", fmt.Sprintf("%v-01-01", year))
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount

				if totalCount != 0 {
					return int(startDate.Month() - 1), nil
				}

				startDate = startDate.AddDate(0, 1, 0)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
}

func (rss *RepositorySearchService) findFirstWeek(language string, stars string, year int, month int) (int, error) {
	startDate, e := time.Parse("2006-01-02", fmt.Sprintf("%d-%02d-%02d", year, month, 1))
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount

				if totalCount != 0 {
					return startDate.AddDate(0, 0, -7).Day(), nil
				}

				startDate = startDate.AddDate(0, 0, 7)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
}

func (rss *RepositorySearchService) findFirstDay(language string, stars string, year int, month int, day int) (int, error) {
	startDate, e := time.Parse("2006-01-02", fmt.Sprintf("%d-%02d-%02d", year, month, day))
	if e != nil {
		return -1, e
	}

	totalCount := -1

	for {
		select {
		case <-time.After(rss.gitHubQueryInterval):
			var (
				queryParameters = fmt.Sprintf("q=language:%s+stars:%s+created:<%s&order=asc",
					language, stars, startDate.Format("2006-01-02"))
				endpoint = fmt.Sprintf("https://api.github.com/search/repositories?%s", queryParameters)
			)

			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Request: %v", endpoint)

			if rss.gitHubPersonalAccessToken != "" {
				req.Header.Set("Authorization", "token "+rss.gitHubPersonalAccessToken)
			}

			resp, err := rss.httpClient.Do(req)
			if err != nil {
				return -1, err
			}

			rss.log.Debugf("Response Status Code: %v", resp.StatusCode)

			if resp.StatusCode == 200 {
				count := Count{}

				body, readError := io.ReadAll(resp.Body)
				if readError != nil {
					return -1, err
				}

				if err = json.Unmarshal(body, &count); err != nil {
					return -1, err
				}

				totalCount = count.TotalCount

				if totalCount != 0 {
					return startDate.AddDate(0, 0, -1).Day(), nil
				}

				startDate = startDate.AddDate(0, 0, 1)
			}

			if err = resp.Body.Close(); err != nil {
				rss.log.Warnf("unable to close response body: %s", err.Error())
			}
		}
	}
}
