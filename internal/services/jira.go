package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"ScrumBoard/internal/models"
)

// JiraService handles all Jira API interactions
type JiraService struct {
	BaseURL  string
	Email    string
	APIToken string
	client   *http.Client
}

// NewJiraService creates a new Jira service instance
func NewJiraService() *JiraService {
	return &JiraService{
		BaseURL:  os.Getenv("JIRA_BASE_URL"),
		Email:    os.Getenv("JIRA_EMAIL"),
		APIToken: os.Getenv("JIRA_API_TOKEN"),
		client:   &http.Client{},
	}
}

// makeRequest makes an authenticated request to Jira API
func (j *JiraService) makeRequest(endpoint string) ([]byte, error) {
	url := fmt.Sprintf("%s/rest/api/3/%s", j.BaseURL, endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Basic auth with email:api_token
	auth := base64.StdEncoding.EncodeToString([]byte(j.Email + ":" + j.APIToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// GetActiveSprint fetches the currently active sprint
func (j *JiraService) GetActiveSprint(boardID string) (*models.Sprint, error) {
	// Use Agile API endpoint for sprints
	url := fmt.Sprintf("%s/rest/agile/1.0/board/%s/sprint?state=active", j.BaseURL, boardID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(j.Email + ":" + j.APIToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Values []models.Sprint `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if len(result.Values) > 0 {
		return &result.Values[0], nil
	}
	return nil, fmt.Errorf("no active sprint found")
}

// GetSprintIssues fetches all issues for a board (filtered by board's filter)
func (j *JiraService) GetSprintIssues(boardID string, sprintID int) ([]models.Story, error) {
	// Use Board API to get issues - this respects the board's filter
	// customfield_10028 = Story Points in this Jira instance
	url := fmt.Sprintf("%s/rest/agile/1.0/board/%s/issue?fields=summary,status,assignee,priority,customfield_10028,labels,created,updated,description&maxResults=200", j.BaseURL, boardID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(j.Email + ":" + j.APIToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Issues []struct {
			ID     string `json:"id"`
			Key    string `json:"key"`
			Fields struct {
				Summary     string `json:"summary"`
				Description any    `json:"description"`
				Status      struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Key string `json:"key"` // done, indeterminate, new
					} `json:"statusCategory"`
				} `json:"status"`
				Assignee *struct {
					DisplayName  string `json:"displayName"`
					EmailAddress string `json:"emailAddress"`
					AvatarUrls   struct {
						Small string `json:"48x48"`
					} `json:"avatarUrls"`
				} `json:"assignee"`
				Priority struct {
					Name string `json:"name"`
				} `json:"priority"`
				StoryPoints float64  `json:"customfield_10028"`
				Labels      []string `json:"labels"`
				Created     string   `json:"created"`
				Updated     string   `json:"updated"`
			} `json:"fields"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	stories := make([]models.Story, 0, len(result.Issues))
	for _, issue := range result.Issues {
		story := models.Story{
			ID:             issue.ID,
			Key:            issue.Key,
			Title:          issue.Fields.Summary,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Key, // done, indeterminate, new
			StoryPoints:    int(issue.Fields.StoryPoints),
			Priority:       issue.Fields.Priority.Name,
			Labels:         issue.Fields.Labels,
			CreatedAt:      issue.Fields.Created,
			UpdatedAt:      issue.Fields.Updated,
		}

		if issue.Fields.Assignee != nil {
			story.Assignee = models.User{
				Name:      issue.Fields.Assignee.DisplayName,
				Email:     issue.Fields.Assignee.EmailAddress,
				AvatarURL: issue.Fields.Assignee.AvatarUrls.Small,
			}
		}

		stories = append(stories, story)
	}

	return stories, nil
}
