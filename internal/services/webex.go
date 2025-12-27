package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// WebexService handles Webex bot messaging
type WebexService struct {
	botToken string
	baseURL  string
}

// WebexMessage represents a message to send
type WebexMessage struct {
	ToPersonEmail string `json:"toPersonEmail,omitempty"`
	ToPersonId    string `json:"toPersonId,omitempty"`
	RoomId        string `json:"roomId,omitempty"`
	Text          string `json:"text,omitempty"`
	Markdown      string `json:"markdown,omitempty"`
}

// WebexResponse represents API response
type WebexResponse struct {
	ID      string `json:"id"`
	RoomId  string `json:"roomId"`
	Created string `json:"created"`
	Error   string `json:"message,omitempty"`
}

// NewWebexService creates a new Webex service
func NewWebexService() *WebexService {
	token := os.Getenv("WEBEX_BOT_TOKEN")
	if token == "" {
		log.Println("⚠️ WEBEX_BOT_TOKEN not set - Webex messaging disabled")
	}
	return &WebexService{
		botToken: token,
		baseURL:  "https://webexapis.com/v1",
	}
}

// IsConfigured checks if Webex bot is configured
func (w *WebexService) IsConfigured() bool {
	return w.botToken != ""
}

// SendStatusPing sends a status request message to a user
func (w *WebexService) SendStatusPing(email, userName, sprintName string, assignedItems []string) error {
	if !w.IsConfigured() {
		return fmt.Errorf("Webex bot not configured - set WEBEX_BOT_TOKEN")
	}

	// Build markdown message
	itemsList := ""
	for _, item := range assignedItems {
		itemsList += fmt.Sprintf("- %s\n", item)
	}

	markdown := fmt.Sprintf(`👋 **Hi %s!**

🏃 **Sprint Update Request**

Your Scrum Master is requesting a status update for **%s**.

📋 **Your assigned items:**
%s
Could you please provide a quick update on the progress?

---
_This is an automated message from Scrum Insights Dashboard_
`, userName, sprintName, itemsList)

	msg := WebexMessage{
		ToPersonEmail: email,
		Markdown:      markdown,
	}

	return w.sendMessage(msg)
}

// SendCustomMessage sends a custom message to a user
func (w *WebexService) SendCustomMessage(email, message string) error {
	if !w.IsConfigured() {
		return fmt.Errorf("Webex bot not configured - set WEBEX_BOT_TOKEN")
	}

	msg := WebexMessage{
		ToPersonEmail: email,
		Markdown:      message,
	}

	return w.sendMessage(msg)
}

// SendInfoRequest sends a specific info request for a story/defect
func (w *WebexService) SendInfoRequest(email, userName, sprintName, itemKey, itemTitle, itemType string) error {
	if !w.IsConfigured() {
		return fmt.Errorf("Webex bot not configured - set WEBEX_BOT_TOKEN")
	}

	typeEmoji := "📖"
	typeName := "User Story"
	if itemType == "defect" {
		typeEmoji = "🐛"
		typeName = "Defect"
	}

	markdown := fmt.Sprintf(`👋 **Hi %s!**

📋 **Info Request**

Your Scrum Master is requesting an update on the following %s:

%s **%s**
> %s

🏃 **Sprint:** %s

Could you please provide a quick update on the progress?

**Please reply to this message with your status update.**

---
_This is an automated message from Scrum Insights Dashboard_
`, userName, typeName, typeEmoji, itemKey, itemTitle, sprintName)

	msg := WebexMessage{
		ToPersonEmail: email,
		Markdown:      markdown,
	}

	return w.sendMessage(msg)
}

// sendMessage sends message via Webex API
func (w *WebexService) sendMessage(msg WebexMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", w.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var webexResp WebexResponse
		json.Unmarshal(body, &webexResp)
		log.Printf("❌ Webex API error: %s", webexResp.Error)
		return fmt.Errorf("Webex API error: %s", webexResp.Error)
	}

	log.Printf("✅ Webex message sent to %s", msg.ToPersonEmail)
	return nil
}
