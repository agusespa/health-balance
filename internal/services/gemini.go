package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"health-balance/internal/database"
	"health-balance/internal/models"
	"health-balance/internal/utils"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func GetHealthSummary(db database.Querier) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	geminiModel := os.Getenv("GEMINI_MODEL_NAME")
	if geminiModel == "" {
		geminiModel = "gemini-3-flash-preview"
	}

	profile, err := db.GetUserProfile()
	if err != nil || profile == nil {
		return "", fmt.Errorf("user profile required for summary")
	}

	scores, err := GetAllWeeklyScores(db)
	if err != nil {
		return "", fmt.Errorf("failed to fetch scores: %v", err)
	}

	if len(scores) == 0 {
		return "No health data available yet to generate a summary. Start tracking your metrics!", nil
	}

	// Use only the last 10 weeks of data for the summary
	limit := min(len(scores), 10)
	recentScores := scores[:limit]

	var weeklyData []WeeklyData
	for _, s := range recentScores {
		h, _ := db.GetHealthMetricsByDate(s.Date)
		f, _ := db.GetFitnessMetricsByDate(s.Date)
		c, _ := db.GetCognitionMetricsByDate(s.Date)

		weeklyData = append(weeklyData, WeeklyData{
			Score:     s,
			Health:    h,
			Fitness:   f,
			Cognition: c,
		})
	}

	prompt := constructPrompt(profile, weeklyData)
	fmt.Println(prompt)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", geminiModel, apiKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed Gemini API call (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

type WeeklyData struct {
	Score     models.MasterScore
	Health    *models.HealthMetrics
	Fitness   *models.FitnessMetrics
	Cognition *models.CognitionMetrics
}

func constructPrompt(profile *models.UserProfile, data []WeeklyData) string {
	age, _ := utils.GetAge(profile, time.Now())

	prompt := fmt.Sprintf(`You are an expert longevity and health coach. Based on the following health data for a %d-year-old %s (height: %.1f cm), provide a summary and actionable recommendations.

The "Master Score" starts at a baseline of 1000 and compounds weekly. It is calculated as follows:

1. **Aging Tax** (Weekly Decay):
   - Formula: (Age^2 / 8000) / 52
   - This rate is applied to the current total score every week, representing natural biological decay.
   - For this user, the weekly decay rate is approximately %.4f%%.

2. **Health Pillar** (Weights):
   - Sleep: +2 points for every point above 75 (Sleep Score 0-100).
   - WHtR (Waist-to-Height Ratio): +/- 10 points for every 0.01 deviation from 0.48 (Goal < 0.48).
   - RHR (Resting Heart Rate): +5 points for every BPM below the 3-month rolling average.
   - Nutrition: +5 points for every 1.0 above 7.0 (Scale 1-10).

3. **Fitness Pillar** (Weights):
   - VO2 Max: +20 points for every 1.0 above the age/sex-adjusted general baseline.
   - Workouts: +20 points for every session above 3 per week.
   - Steps: +1 point for every 150 steps above 8,000 per day.
   - Mobility: +10 points for every session above 3 per week.
   - Cardio Recovery: +3 points for every BPM drop above 25 (60s post-exercise).

4. **Cognition Pillar** (Weights):
   - Memory (Dual N-Back): +20 points for every level above Level 2.
   - Reaction Time: +1 point for every 2ms improvement over the age-adjusted baseline.
   - Mindfulness: +5 points for every session above 3 per week.
   - Deep Learning (Language/Instruments): +2 points for every 10 minutes above 90 minutes/week.

Detailed Weekly Data (most recent first):
`, age, profile.Sex, profile.HeightCm, (float64(age*age)/8000.0)/52.0*100.0)

	for _, d := range data {
		s := d.Score
		prompt += fmt.Sprintf("\n### Week of %s\n", s.Date)
		prompt += fmt.Sprintf("- **Scores**: Total: %.1f | Health: %.1f | Fitness: %.1f | Cognition: %.1f | Aging Tax: -%.1f\n",
			s.Score, s.HealthScore, s.FitnessScore, s.CognitionScore, s.AgingTax)

		if d.Health != nil {
			h := d.Health
			prompt += fmt.Sprintf("- **Health Metrics**: Sleep Score: %d | Waist: %.1f cm | RHR: %d bpm | Nutrition: %.1f/10\n",
				h.SleepScore, h.WaistCm, h.RHR, h.NutritionScore)
		}
		if d.Fitness != nil {
			f := d.Fitness
			prompt += fmt.Sprintf("- **Fitness Metrics**: VO2 Max: %.1f | Workouts: %d | Daily Steps: %d | Mobility: %d | Cardio Recovery: %d bpm drop\n",
				f.VO2Max, f.Workouts, f.DailySteps, f.Mobility, f.CardioRecovery)
		}
		if d.Cognition != nil {
			c := d.Cognition
			prompt += fmt.Sprintf("- **Cognition Metrics**: Dual N-Back Level: %d | Reaction Time: %d ms | Mindfulness: %d sessions | Deep Learning: %d total minutes\n",
				c.DualNBackLevel, c.ReactionTime, c.Mindfulness, c.DeepLearning)
		}
	}

	prompt += "\nAnalysis Task:\n"
	prompt += "- Identify the primary bottlenecks for their longevity score by looking at the raw metrics, not just the scores.\n"
	prompt += "- Provide 3-5 specific, high-impact recommendations tailored to the weights above.\n"
	prompt += "- Keep it very concise, data-driven and clinical."
	prompt += "- Format your response using clean Markdown with bold headers and bullet points. Avoid nested bullet points and complex formatting."

	return prompt
}
