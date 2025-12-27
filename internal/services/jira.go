package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

// GetBoards fetches all boards accessible to the user
func (j *JiraService) GetBoards() ([]models.Board, error) {
	url := fmt.Sprintf("%s/rest/agile/1.0/board?maxResults=100", j.BaseURL)

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
		Values []models.Board `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Values, nil
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

// GetAllSprints fetches all sprints that have AMFDEVFT bugs (board-independent)
func (j *JiraService) GetAllSprints(boardID string) ([]models.Sprint, error) {
	// Fetch all AMFDEVFT bugs to get unique sprints
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", j.BaseURL)

	requestBody := map[string]interface{}{
		"jql":        "type=Bug AND labels=AMFDEVFT AND sprint is not EMPTY",
		"maxResults": 1000,
		"fields":     []string{"customfield_10020"}, // Sprint field
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
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
			Fields struct {
				Sprint []models.Sprint `json:"customfield_10020"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Extract unique sprints
	sprintMap := make(map[int]models.Sprint)
	for _, issue := range result.Issues {
		for _, sprint := range issue.Fields.Sprint {
			sprintMap[sprint.ID] = sprint
		}
	}

	// Convert map to slice
	sprints := make([]models.Sprint, 0, len(sprintMap))
	for _, sprint := range sprintMap {
		sprints = append(sprints, sprint)
	}

	return sprints, nil
}

// GetSprintByID fetches a specific sprint by ID
func (j *JiraService) GetSprintByID(sprintID int) (models.Sprint, error) {
	url := fmt.Sprintf("%s/rest/agile/1.0/sprint/%d", j.BaseURL, sprintID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.Sprint{}, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(j.Email + ":" + j.APIToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return models.Sprint{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Sprint{}, err
	}

	var sprint models.Sprint
	if err := json.Unmarshal(data, &sprint); err != nil {
		return models.Sprint{}, err
	}

	return sprint, nil
}

// GetStoriesByEngineerAndSprint fetches stories for a specific engineer in a sprint
func (j *JiraService) GetStoriesByEngineerAndSprint(engineerEmail string, sprintID int) ([]models.Story, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", j.BaseURL)

	jql := fmt.Sprintf("type=Story AND assignee='%s' AND sprint=%d ORDER BY created DESC", engineerEmail, sprintID)

	requestBody := map[string]interface{}{
		"jql":        jql,
		"maxResults": 200,
		"fields":     []string{"summary", "status", "assignee", "priority", "customfield_10028", "labels", "created", "updated", "description"},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
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
			Key    string `json:"key"`
			Fields struct {
				Summary     string      `json:"summary"`
				Description interface{} `json:"description"` // Can be string or object
				Status      struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Key string `json:"key"`
					} `json:"statusCategory"`
				} `json:"status"`
				Assignee struct {
					DisplayName  string `json:"displayName"`
					EmailAddress string `json:"emailAddress"`
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
		// Extract description as string if possible
		desc := ""
		if issue.Fields.Description != nil {
			if descStr, ok := issue.Fields.Description.(string); ok {
				desc = descStr
			}
		}

		story := models.Story{
			Key:            issue.Key,
			Title:          issue.Fields.Summary,
			Description:    desc,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Key,
			Assignee: models.User{
				Name:  issue.Fields.Assignee.DisplayName,
				Email: issue.Fields.Assignee.EmailAddress,
			},
			Priority:    issue.Fields.Priority.Name,
			StoryPoints: int(issue.Fields.StoryPoints),
			Labels:      issue.Fields.Labels,
			CreatedAt:   issue.Fields.Created,
			UpdatedAt:   issue.Fields.Updated,
		}
		stories = append(stories, story)
	}

	log.Printf("📖 Found %d stories for %s in sprint %d", len(stories), engineerEmail, sprintID)
	return stories, nil
}

// GetDefectsByEngineerAndSprint fetches defects for a specific engineer in a sprint
func (j *JiraService) GetDefectsByEngineerAndSprint(engineerEmail string, sprintID int) ([]models.Defect, error) {
	jql := fmt.Sprintf("type=Bug AND assignee='%s' AND sprint=%d ORDER BY created DESC", engineerEmail, sprintID)

	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", j.BaseURL)

	requestBody := map[string]interface{}{
		"jql":        jql,
		"maxResults": 200,
		"fields":     []string{"summary", "description", "status", "assignee", "reporter", "priority", "labels", "created", "updated", "resolutiondate", "customfield_10031", "environment", "versions"},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
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
				Summary     string      `json:"summary"`
				Description interface{} `json:"description"` // Can be string or object
				Status      struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Key string `json:"key"`
					} `json:"statusCategory"`
				} `json:"status"`
				Assignee *struct {
					DisplayName  string `json:"displayName"`
					EmailAddress string `json:"emailAddress"`
				} `json:"assignee"`
				Reporter *struct {
					DisplayName  string `json:"displayName"`
					EmailAddress string `json:"emailAddress"`
				} `json:"reporter"`
				Priority struct {
					Name string `json:"name"`
				} `json:"priority"`
				Severity        string      `json:"customfield_10031"`
				Labels          []string    `json:"labels"`
				Created         string      `json:"created"`
				Updated         string      `json:"updated"`
				ResolutionDate  *string     `json:"resolutiondate"`
				Environment     interface{} `json:"environment"` // Can be string or object
				AffectedVersion []struct {
					Name string `json:"name"`
				} `json:"versions"`
			} `json:"fields"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	defects := make([]models.Defect, 0, len(result.Issues))
	for _, issue := range result.Issues {
		// Extract description as string if possible
		desc := ""
		if issue.Fields.Description != nil {
			if descStr, ok := issue.Fields.Description.(string); ok {
				desc = descStr
			}
		}

		// Extract environment as string if possible
		env := ""
		if issue.Fields.Environment != nil {
			if envStr, ok := issue.Fields.Environment.(string); ok {
				env = envStr
			}
		}

		defect := models.Defect{
			ID:             issue.ID,
			Key:            issue.Key,
			Summary:        issue.Fields.Summary,
			Description:    desc,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Key,
			Priority:       issue.Fields.Priority.Name,
			Severity:       issue.Fields.Severity,
			Labels:         issue.Fields.Labels,
			Environment:    env,
		}

		if issue.Fields.Assignee != nil {
			defect.Assignee = models.User{
				Name:  issue.Fields.Assignee.DisplayName,
				Email: issue.Fields.Assignee.EmailAddress,
			}
		}

		if issue.Fields.Reporter != nil {
			defect.Reporter = models.User{
				Name:  issue.Fields.Reporter.DisplayName,
				Email: issue.Fields.Reporter.EmailAddress,
			}
		}

		if len(issue.Fields.AffectedVersion) > 0 {
			defect.AffectedVersion = issue.Fields.AffectedVersion[0].Name
		}

		defects = append(defects, defect)
	}

	log.Printf("🐛 Found %d defects for %s in sprint %d", len(defects), engineerEmail, sprintID)
	return defects, nil
}

// GetSprintIssuesByJQL fetches AMF team issues using direct JQL search (no board number needed)
func (j *JiraService) GetSprintIssuesByJQL(sprintID int) ([]models.Story, error) {
	// Use JQL to search across ALL boards for sprint + AMF team filter
	// Try multiple JQL variations to find the right syntax

	// First try: Use "Team" field name
	jql := fmt.Sprintf("sprint = %d AND Team ~ \"AMF:*\" ORDER BY created DESC", sprintID)

	log.Printf("🔍 JQL Query: %s", jql)

	url := fmt.Sprintf("%s/rest/api/3/search/jql", j.BaseURL)

	requestBody := map[string]interface{}{
		"jql":        jql,
		"maxResults": 500,
		"fields":     []string{"summary", "status", "assignee", "priority", "customfield_10028", "customfield_10001", "labels", "created", "updated", "description"},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
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

	if resp.StatusCode != 200 {
		log.Printf("❌ Jira API error: Status %d, Response: %s", resp.StatusCode, string(data))
		return nil, fmt.Errorf("jira API returned status %d", resp.StatusCode)
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
				Team *struct {
					Name string `json:"name"`
				} `json:"customfield_10001"` // Team field
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
	teamMap := make(map[string]int)

	// Log first issue for debugging
	if len(result.Issues) > 0 {
		firstIssue := result.Issues[0]
		log.Printf("🔬 Sample issue %s:", firstIssue.Key)
		if firstIssue.Fields.Team != nil {
			log.Printf("   Team: %s", firstIssue.Fields.Team.Name)
		}
		if firstIssue.Fields.Assignee != nil {
			log.Printf("   Assignee: %s", firstIssue.Fields.Assignee.DisplayName)
		}
	}

	for _, issue := range result.Issues {
		// Track team names
		if issue.Fields.Team != nil {
			teamMap[issue.Fields.Team.Name]++
		}

		story := models.Story{
			ID:             issue.ID,
			Key:            issue.Key,
			Title:          issue.Fields.Summary,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Key,
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

	// Log results
	log.Printf("🔍 Sprint %d: Found %d AMF team issues via JQL", sprintID, len(stories))
	if len(teamMap) > 0 {
		log.Printf("👥 AMF Teams:")
		for team, count := range teamMap {
			log.Printf("   - %s: %d issues", team, count)
		}
	}

	return stories, nil
}

// GetSprintIssues fetches all issues for a specific sprint filtered by team name containing "AMF"
func (j *JiraService) GetSprintIssues(boardID string, sprintID int) ([]models.Story, error) {
	// Use Board + Sprint API to get issues for a specific sprint
	// customfield_10028 = Story Points
	// customfield_10001 = Team field (contains team name like "AMF: Interstellar", "AMF: Avengers")
	url := fmt.Sprintf("%s/rest/agile/1.0/board/%s/sprint/%d/issue?fields=summary,status,assignee,priority,customfield_10028,customfield_10001,labels,created,updated,description&maxResults=500", j.BaseURL, boardID, sprintID)

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
				Team *struct {
					Name string `json:"name"`
				} `json:"customfield_10001"` // Team field
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
	teamMap := make(map[string]int) // Track unique teams

	// Log first issue for debugging
	if len(result.Issues) > 0 {
		firstIssue := result.Issues[0]
		log.Printf("🔬 Sample issue %s:", firstIssue.Key)
		if firstIssue.Fields.Team != nil {
			log.Printf("   Team: %s", firstIssue.Fields.Team.Name)
		} else {
			log.Printf("   Team: <none>")
		}
		if firstIssue.Fields.Assignee != nil {
			log.Printf("   Assignee: %s", firstIssue.Fields.Assignee.DisplayName)
		}
	}

	for _, issue := range result.Issues {
		// Filter: Only include issues with AMF in team name
		hasAMF := false
		if issue.Fields.Team != nil {
			teamName := issue.Fields.Team.Name
			teamMap[teamName]++
			if strings.Contains(strings.ToUpper(teamName), "AMF") {
				hasAMF = true
			}
		}

		// Skip non-AMF issues
		if !hasAMF {
			continue
		}

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

	// Log results
	log.Printf("🔍 Board %s, Sprint %d: Found %d AMF team issues", boardID, sprintID, len(stories))
	if len(teamMap) > 0 {
		log.Printf("👥 AMF Teams:")
		for team, count := range teamMap {
			if strings.Contains(strings.ToUpper(team), "AMF") {
				log.Printf("   - %s: %d issues", team, count)
			}
		}
	}

	return stories, nil
}

// DiscoverAMFBoards scans board IDs in parallel to find boards with AMF team issues for a sprint
func (j *JiraService) DiscoverAMFBoards(sprintID int, startID int, endID int) []string {
	log.Printf("🔍 Discovering AMF boards for sprint %d (scanning boards %d to %d in parallel)...", sprintID, startID, endID)

	type result struct {
		boardID string
		count   int
	}

	results := make(chan result, 100)
	var wg sync.WaitGroup

	// Scan in parallel with 50 concurrent workers
	semaphore := make(chan struct{}, 50)

	for boardID := startID; boardID <= endID; boardID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Try to fetch issues from this board
			stories, err := j.GetSprintIssues(fmt.Sprintf("%d", id), sprintID)
			if err != nil {
				return // Skip boards that error out
			}

			// If we found AMF issues, send result
			if len(stories) > 0 {
				results <- result{boardID: fmt.Sprintf("%d", id), count: len(stories)}
			}
		}(boardID)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	amfBoards := make([]string, 0)
	for res := range results {
		amfBoards = append(amfBoards, res.boardID)
		log.Printf("✅ Found board %s with %d AMF issues", res.boardID, res.count)
	}

	log.Printf("🎯 Discovery complete! Found %d boards with AMF data: %v", len(amfBoards), amfBoards)
	return amfBoards
}

// GetSprintDefects fetches all bugs/defects for a board in the current sprint
func (j *JiraService) GetSprintDefects(boardID string, sprintID int) ([]models.Defect, error) {
	// JQL: Bugs with AMFDEVFT label in the specified sprint
	jql := fmt.Sprintf("type=Bug AND labels=AMFDEVFT AND sprint=%d", sprintID)

	// Use direct JQL search API instead of board API to bypass board filters
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", j.BaseURL)

	// Create JSON request body
	requestBody := map[string]interface{}{
		"jql":        jql,
		"maxResults": 200,
		"fields":     []string{"summary", "description", "status", "assignee", "reporter", "priority", "labels", "created", "updated", "resolution", "resolutiondate", "environment", "versions", "customfield_10032"},
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
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
				Reporter *struct {
					DisplayName  string `json:"displayName"`
					EmailAddress string `json:"emailAddress"`
					AvatarUrls   struct {
						Small string `json:"48x48"`
					} `json:"avatarUrls"`
				} `json:"reporter"`
				Priority struct {
					Name string `json:"name"`
				} `json:"priority"`
				Labels         []string `json:"labels"`
				Created        string   `json:"created"`
				Updated        string   `json:"updated"`
				ResolutionDate *string  `json:"resolutiondate"`
				Environment    any      `json:"environment"`
				Versions       []struct {
					Name string `json:"name"`
				} `json:"versions"`
				Severity string `json:"customfield_10032"` // Custom field for severity
			} `json:"fields"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	defects := make([]models.Defect, 0, len(result.Issues))
	for _, issue := range result.Issues {
		defect := models.Defect{
			ID:             issue.ID,
			Key:            issue.Key,
			Summary:        issue.Fields.Summary,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Key,
			Priority:       issue.Fields.Priority.Name,
			Labels:         issue.Fields.Labels,
		}

		// Parse created date
		if createdTime, err := parseJiraDate(issue.Fields.Created); err == nil {
			defect.CreatedAt = createdTime
			defect.AgeInDays = int(time.Since(createdTime).Hours() / 24)
		}

		// Parse updated date
		if updatedTime, err := parseJiraDate(issue.Fields.Updated); err == nil {
			defect.UpdatedAt = updatedTime
		}

		// Parse resolution date
		if issue.Fields.ResolutionDate != nil && *issue.Fields.ResolutionDate != "" {
			if resolvedTime, err := parseJiraDate(*issue.Fields.ResolutionDate); err == nil {
				defect.ResolvedAt = &resolvedTime
				defect.ResolutionTime = int(resolvedTime.Sub(defect.CreatedAt).Hours())
			}
		}

		// Assignee
		if issue.Fields.Assignee != nil {
			defect.Assignee = models.User{
				Name:      issue.Fields.Assignee.DisplayName,
				Email:     issue.Fields.Assignee.EmailAddress,
				AvatarURL: issue.Fields.Assignee.AvatarUrls.Small,
			}
		}

		// Reporter
		if issue.Fields.Reporter != nil {
			defect.Reporter = models.User{
				Name:      issue.Fields.Reporter.DisplayName,
				Email:     issue.Fields.Reporter.EmailAddress,
				AvatarURL: issue.Fields.Reporter.AvatarUrls.Small,
			}
		}

		// Environment
		if issue.Fields.Environment != nil {
			if envStr, ok := issue.Fields.Environment.(string); ok {
				defect.Environment = envStr
			}
		}

		// Affected version
		if len(issue.Fields.Versions) > 0 {
			defect.AffectedVersion = issue.Fields.Versions[0].Name
		}

		// Severity - map from custom field or derive from priority
		defect.Severity = deriveSeverity(issue.Fields.Severity, issue.Fields.Priority.Name)

		defects = append(defects, defect)
	}

	return defects, nil
}

// parseJiraDate parses Jira date format
func parseJiraDate(dateStr string) (time.Time, error) {
	// Jira uses ISO 8601 format: 2024-01-15T10:30:00.000+0000
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// deriveSeverity derives severity from custom field or priority
func deriveSeverity(customSeverity, priority string) string {
	if customSeverity != "" {
		return customSeverity
	}

	// Map priority to severity if custom field is not available
	switch priority {
	case "Blocker", "Highest":
		return "Critical"
	case "Critical", "High":
		return "High"
	case "Major", "Medium":
		return "Medium"
	default:
		return "Low"
	}
}
