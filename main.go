package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// --- Types ---

type TemplateData struct {
	OnThisDay  OnThisDay
	LeetCode   LeetCode
	Timestamp  string
	GitHubUser string
}

type OnThisDay struct {
	Available bool
	Text      string
	Year      int
	PageURL   string
}

type LeetCode struct {
	Available  bool
	Title      string
	Difficulty string
	URL        string
}

// --- Wikipedia: On This Day ---

func fetchOnThisDay() OnThisDay {
	now := time.Now().UTC()
	url := fmt.Sprintf("https://en.wikipedia.org/api/rest_v1/feed/onthisday/selected/%02d/%02d", now.Month(), now.Day())

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		log.Printf("WARN: on this day request error: %v", err)
		return OnThisDay{}
	}
	req.Header.Set("User-Agent", "dannylongeuay-profile-readme/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("WARN: on this day fetch error: %v", err)
		return OnThisDay{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("WARN: on this day API returned %d", resp.StatusCode)
		return OnThisDay{}
	}

	var result struct {
		Selected []struct {
			Text  string `json:"text"`
			Year  int    `json:"year"`
			Pages []struct {
				Content struct {
					Desktop struct {
						Page string `json:"page"`
					} `json:"desktop"`
				} `json:"content_urls"`
			} `json:"pages"`
		} `json:"selected"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("WARN: on this day decode error: %v", err)
		return OnThisDay{}
	}

	if len(result.Selected) == 0 {
		return OnThisDay{}
	}

	// Pick a random-ish event based on the day
	idx := now.YearDay() % len(result.Selected)
	event := result.Selected[idx]
	if event.Text == "" {
		return OnThisDay{}
	}

	pageURL := ""
	if len(event.Pages) > 0 {
		pageURL = event.Pages[0].Content.Desktop.Page
	}

	return OnThisDay{
		Available: true,
		Text:      event.Text,
		Year:      event.Year,
		PageURL:   pageURL,
	}
}

// --- LeetCode: Daily Challenge ---

func fetchLeetCodeDaily() LeetCode {
	query := `{"query":"query { activeDailyCodingChallengeQuestion { link question { title difficulty } } }"}`

	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://leetcode.com/graphql", strings.NewReader(query))
	if err != nil {
		log.Printf("WARN: leetcode request error: %v", err)
		return LeetCode{}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("WARN: leetcode fetch error: %v", err)
		return LeetCode{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("WARN: leetcode API returned %d", resp.StatusCode)
		return LeetCode{}
	}

	var result struct {
		Data struct {
			ActiveDailyCodingChallengeQuestion struct {
				Link     string `json:"link"`
				Question struct {
					Title      string `json:"title"`
					Difficulty string `json:"difficulty"`
				} `json:"question"`
			} `json:"activeDailyCodingChallengeQuestion"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("WARN: leetcode decode error: %v", err)
		return LeetCode{}
	}

	q := result.Data.ActiveDailyCodingChallengeQuestion
	if q.Question.Title == "" {
		return LeetCode{}
	}

	return LeetCode{
		Available:  true,
		Title:      q.Question.Title,
		Difficulty: q.Question.Difficulty,
		URL:        "https://leetcode.com" + q.Link,
	}
}

// --- Main ---

func main() {
	user := "dannylongeuay"

	tmpl, err := template.ParseFiles("readme.tmpl")
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	data := TemplateData{
		GitHubUser: user,
		Timestamp:  time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(2)

	go func() {
		defer wg.Done()
		otd := fetchOnThisDay()
		mu.Lock()
		data.OnThisDay = otd
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		lc := fetchLeetCodeDaily()
		mu.Lock()
		data.LeetCode = lc
		mu.Unlock()
	}()

	wg.Wait()

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	if err := os.WriteFile("README.md", buf.Bytes(), 0644); err != nil {
		log.Fatalf("failed to write README.md: %v", err)
	}

	log.Println("README.md updated successfully")
}
