package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type SlackNotifier struct {
	webhookURL string
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	if webhookURL == "" {
		webhookURL = os.Getenv("SLACK_WEBHOOK")
	}
	return &SlackNotifier{webhookURL: webhookURL}
}

func (s *SlackNotifier) NotifyCompromised(pkg, version, reason string) error {
	if s.webhookURL == "" {
		return fmt.Errorf("no Slack webhook configured")
	}

	payload := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": "🚨 Compromised Package Detected",
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Package:*\n%s@%s", pkg, version)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Reason:*\n%s", reason)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Detected:*\n%s", time.Now().Format("2006-01-02 15:04 UTC"))},
					{"type": "mrkdwn", "text": "*Action:*\nBlocked by SafeUpgrade policy"},
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "⚡ *Immediate action required:* Check if any repos are running this version.",
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Post(s.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("slack returned %d", resp.StatusCode)
	}
	return nil
}

func (s *SlackNotifier) NotifyUpgradeComplete(repo string, upgraded, skipped int, prURL string) error {
	if s.webhookURL == "" {
		return nil
	}

	text := fmt.Sprintf("✅ *SafeUpgrade complete* for `%s`\n• %d upgraded, %d skipped", repo, upgraded, skipped)
	if prURL != "" {
		text += fmt.Sprintf("\n• PR: %s", prURL)
	}

	payload := map[string]string{"text": text}
	body, _ := json.Marshal(payload)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Post(s.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
