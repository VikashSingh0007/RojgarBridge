package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"rojgarbridge-backend/internal/models"
)

type Notifier interface {
	NotifyJobPublished(job *models.Job) error
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

func (t *TelegramNotifier) NotifyJobPublished(job *models.Job) error {
	if !t.IsEnabled() {
		return nil
	}

	vacanciesStr := "Refer Notification"
	if job.Details.TotalVacancies > 0 {
		vacanciesStr = fmt.Sprintf("%d Posts", job.Details.TotalVacancies)
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
		job.Details.Organization,
		job.Details.Title,
		job.Details.Category,
		job.Details.State,
		vacanciesStr,
		job.Details.Qualification,
		job.Details.ApplicationEnd,
		job.Details.OfficialApplyURL,
	)

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       msgText,
		"parse_mode": "Markdown",
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
