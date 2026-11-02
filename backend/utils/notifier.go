package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type NotificationJob struct {
	Title            string
	Organization     string
	Category         string
	State            string
	TotalVacancies   int
	Qualification    string
	ApplicationEnd   string
	OfficialApplyURL string
}

type Notifier interface {
	NotifyJobPublished(job NotificationJob) error
	IsEnabled() bool
}

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (t *TelegramNotifier) IsEnabled() bool {
	return t.botToken != "" && t.chatID != ""
}

func (t *TelegramNotifier) NotifyJobPublished(job NotificationJob) error {
	if !t.IsEnabled() {
		return nil
	}

	vacanciesStr := "Refer Notification"
	if job.TotalVacancies > 0 {
		vacanciesStr = fmt.Sprintf("%d Posts", job.TotalVacancies)
	}

	msgText := fmt.Sprintf(`📢 *NEW SARKARI NAUKRI ALERT*

🏛️ *%s*
📌 *Title:* %s
📂 *Category:* %s
📍 *Location:* %s
👥 *Vacancies:* %s
🎓 *Eligibility:* %s
⏰ *Last Date:* %s

🔗 [Official Notification / Apply Online](%s)

_Shared by RojgarHub Official Engine_`,
		job.Organization,
		job.Title,
		job.Category,
		job.State,
		vacanciesStr,
		job.Qualification,
		job.ApplicationEnd,
		job.OfficialApplyURL,
	)

	payload := map[string]interface{}{
		"chat_id":                  t.chatID,
		"text":                     msgText,
		"parse_mode":               "Markdown",
		"disable_web_page_preview": false,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := t.client.Post(apiURL, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Printf("[Telegram Error] Failed to send alert: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Telegram Warning] Status %d from Telegram API", resp.StatusCode)
	}

	return nil
}
