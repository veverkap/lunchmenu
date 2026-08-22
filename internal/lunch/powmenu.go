package lunch

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiBase = "https://powhatancounty.api.nutrislice.com/menu/api/weeks/school"
)

type MenuResponse struct {
	StartDate  string `json:"start_date"`
	MenuTypeID int    `json:"menu_type_id"`
	Days       []Day  `json:"days"`
}

type Day struct {
	Date               string         `json:"date"`
	HasUnpublishedMenu bool           `json:"has_unpublished_menus"`
	MenuInfo           map[string]any `json:"menu_info"`
	MenuItems          []MenuItem     `json:"menu_items"`
}

type MenuItem struct {
	Text           string `json:"text"`
	Position       int    `json:"position"`
	IsSectionTitle bool   `json:"is_section_title"`
	Bold           bool   `json:"bold"`
	Featured       bool   `json:"featured"`
	Food           *Food  `json:"food"`
}

type Food struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func main() {
	school := flag.String("school", "", "School name (e.g., flat-rock-elementary)")
	dateStr := flag.String("date", "", "Date in YYYY-MM-DD format (defaults to tomorrow)")
	telegramToken := flag.String("token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token")
	telegramChat := flag.String("chat", os.Getenv("TELEGRAM_CHAT_ID"), "Telegram chat ID")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if *school == "" {
		log.Fatal("school flag is required")
	}

	// Determine the target date
	var targetDate time.Time
	if *dateStr != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", *dateStr)
		if err != nil {
			log.Fatalf("Invalid date format: %v", err)
		}
	} else {
		// Use tomorrow
		targetDate = time.Now().AddDate(0, 0, 1)
	}

	// Fetch the menu
	menu, err := fetchMenu(*school, targetDate)
	if err != nil {
		log.Fatalf("Failed to fetch menu: %v", err)
	}

	// Format the menu
	formatted := formatMenu(*school, targetDate, menu)

	if *telegramToken != "" && *telegramChat != "" {
		if err := sendTelegram(*telegramToken, *telegramChat, formatted); err != nil {
			log.Fatalf("Failed to send Telegram message: %v", err)
		}
		fmt.Println("Message sent to Telegram")
	} else {
		fmt.Println(formatted)
	}
}

func fetchMenu(school string, date time.Time) (*MenuResponse, error) {
	url := fmt.Sprintf(
		"%s/%s/menu-type/lunch/%04d/%02d/%02d/",
		apiBase,
		school,
		date.Year(),
		date.Month(),
		date.Day(),
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var menu MenuResponse
	if err := json.NewDecoder(resp.Body).Decode(&menu); err != nil {
		return nil, err
	}

	return &menu, nil
}

func formatMenu(school string, date time.Time, menu *MenuResponse) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "*Menu for %s*\n", date.Format("Monday, January 2, 2006"))
	fmt.Fprintf(&sb, "School: %s\n\n", school)

	dateStr := date.Format("2006-01-02")
	for _, day := range menu.Days {
		if day.Date != dateStr {
			continue
		}

		if len(day.MenuItems) == 0 {
			fmt.Fprint(&sb, "No menu items available")
			break
		}

		for _, item := range day.MenuItems {
			if item.IsSectionTitle {
				fmt.Fprintf(&sb, "\n*%s*\n", item.Text)
			} else if item.Food != nil && item.Food.Name != "" {
				fmt.Fprintf(&sb, "• %s\n", item.Food.Name)
			} else if item.Text != "" {
				fmt.Fprintf(&sb, "• %s\n", item.Text)
			}
		}
	}

	return sb.String()
}

func sendTelegram(token, chatID, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := map[string]any{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Telegram API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
