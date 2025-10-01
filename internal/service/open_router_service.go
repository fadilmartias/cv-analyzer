package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fadilmartias/cv-analyzer/internal/config"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type OpenRouterServiceInterface interface {
	Evaluate(cv, report string) (int, any, error)
}

type OpenRouterService struct {
	APIKey string
}

func NewOpenRouterService() *OpenRouterService {
	return &OpenRouterService{
		APIKey: config.LoadOpenRouterConfig().APIKey,
	}
}

func (s *OpenRouterService) Evaluate(cv, report string) (int, any, error) {
	// Minta jawaban dalam format JSON biar aman diparse
	jobDescription := `You'll be building new product features alongside a frontend engineer and product manager using our Agile methodology, as well as addressing issues to ensure our apps are robust and our codebase is clean. As a Product Engineer, you'll write clean, efficient code to enhance our product's codebase in meaningful ways.

In addition to classic backend work, this role also touches on building AI-powered systems, where you’ll design and orchestrate how large language models (LLMs) integrate into Rakamin’s product ecosystem.

Here are some real examples of the work in our team:

Collaborating with frontend engineers and 3rd parties to build robust backend solutions that support highly configurable platforms and cross-platform integration.
Developing and maintaining server-side logic for central database, ensuring high performance throughput and response time.
Designing and fine-tuning AI prompts that align with product requirements and user contexts.
Building LLM chaining flows, where the output from one model is reliably passed to and enriched by another.
Implementing Retrieval-Augmented Generation (RAG) by embedding and retrieving context from vector databases, then injecting it into AI prompts to improve accuracy and relevance.
Handling long-running AI processes gracefully — including job orchestration, async background workers, and retry mechanisms.
Designing safeguards for uncontrolled scenarios: managing failure cases from 3rd party APIs and mitigating the randomness/nondeterminism of LLM outputs.
Leveraging AI tools and workflows to increase team productivity (e.g., AI-assisted code generation, automated QA, internal bots).
Writing reusable, testable, and efficient code to improve the functionality of our existing systems.
Strengthening our test coverage with RSpec to build robust and reliable web apps.
Conducting full product lifecycles, from idea generation to design, implementation, testing, deployment, and maintenance.
Providing input on technical feasibility, timelines, and potential product trade-offs, working with business divisions.
Actively engaging with users and stakeholders to understand their needs and translate them into backend and AI-driven improvements.


Required qualification

We're looking for candidates with a strong track record of working on backend technologies of web apps, ideally with exposure to AI/LLM development or a strong desire to learn.

You should have experience with backend languages and frameworks (Node.js, Django, Rails), as well as modern backend tooling and technologies such as:

Database management (MySQL, PostgreSQL, MongoDB)
RESTful APIs
Security compliance
Cloud technologies (AWS, Google Cloud, Azure)
Server-side languages (Java, Python, Ruby, or JavaScript)
Understanding of frontend technologies
User authentication and authorization between multiple systems, servers, and environments
Scalable application design principles
Creating database schemas that represent and support business processes
Implementing automated testing platforms and unit tests
Familiarity with LLM APIs, embeddings, vector databases and prompt design best practices
We're not big on credentials, so a Computer Science degree or graduating from a prestigious university isn't something we emphasize. We care about what you can do and how you do it, not how you got here.

While you'll report to a CTO directly, Rakamin is a company where Managers of One thrive. We're quick to trust that you can do it, and here to support you. You can expect to be counted on and do your best work and build a career here.

This is a remote job. You're free to work where you work best: home office, co-working space, coffee shops. To ensure time zone overlap with our current team and maintain well communication, we're only looking for people based in Indonesia.`

	prompt := fmt.Sprintf(`
You are an AI evaluator for a Product Engineer (Backend).
This is job vacancy description for this role:
%s
Evaluate the candidate's CV and Project Report based on job vacancy description above and the criteria below.

Return your answer STRICTLY in JSON format with this schema:
{
	"cv_match_rate": <float with 2 decimal places, range 0-1 based on cv breakdown score>,
	"cv_feedback": "<feedback about CV>",
	"project_score": <float with 2 decimal places, range 0-10 based on project breakdown score>,
	"project_feedback": "<feedback about Project Report>",
	"overall_summary": "<summary of overall impression, strengths, and areas to improve>",
  "breakdown": {
    "cv": {
	"technical_skills_match": <number 1-5, criteria: backend, databases, APIs, cloud, and AI/LLM exposure>,
	"experience_level": <number 1-5, criteria: years, project complexity>,
	"relevant_achievements": <number 1-5, criteria: impact, scale>,
	"cultural_fit": <number 1-5, criteria: communication, learning attitude>,
	},
    "project_report": {
      "correctness": <number 1-5, criteria: prompt design, chaining, RAG, handling errors>,
      "code_quality": <number 1-5, criteria: clean, modular, testable>,
      "resilience": <number 1-5, criteria: handles failures, retries>,
      "documentation": <number 1-5, criteria: clear README, explanation of trade-offs>,
      "creativity_or_bonus": <number 1-5, criteria: optional improvements like authentication, deployment, dashboards, etc.>
    }
  },
  
}

CV:
%s

Project Report:
%s
`, jobDescription, cv, report)

	payload := map[string]any{
		"model": "openai/gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an AI evaluating job applications for Product Engineer (Backend)."},
			{"role": "user", "content": prompt},
		},
	}
	body, _ := json.Marshal(payload)

	log.Printf("LLM request: %s", string(body))

	log.Printf("LLM API Key: %s", s.APIKey)

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	log.Printf("LLM response: %s", string(respBody))

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return 0, "", err
	}

	if len(parsed.Choices) == 0 {
		return 0, "No feedback", nil
	}

	type EvaluationResult struct {
		OverallScore int `json:"overall_score"`
		Breakdown    struct {
			TechnicalBackend struct {
				Score   int    `json:"score"`
				Comment string `json:"comment"`
			} `json:"technical_backend"`
			ProductMindset struct {
				Score   int    `json:"score"`
				Comment string `json:"comment"`
			} `json:"product_mindset"`
			CaseStudyQuality struct {
				Score   int    `json:"score"`
				Comment string `json:"comment"`
			} `json:"case_study_quality"`
			Communication struct {
				Score   int    `json:"score"`
				Comment string `json:"comment"`
			} `json:"communication"`
		} `json:"breakdown"`
		FinalFeedback string `json:"final_feedback"`
	}

	var result EvaluationResult
	// Parse JSON dari isi jawaban LLM

	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &result); err != nil {
		// fallback kalau gagal parse
		return 0, parsed.Choices[0].Message.Content, nil
	}

	return result.OverallScore, result.Breakdown, nil
}

func (s *OpenRouterService) Evaluate2(cv, report string) (string, int, error) {
	prompt := fmt.Sprintf(`
Please evaluate the following job application.
Return your answer STRICTLY in JSON format:
{
  "score": <number 0-100>,
  "feedback": "<short feedback text>"
}

CV:
%s

Report:
%s
`, cv, report)

	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+s.APIKey).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": "openai/gpt-4o-mini",
			"messages": []map[string]string{
				{"role": "system", "content": "You are an AI evaluating job applications."},
				{"role": "user", "content": prompt},
			},
		}).
		Post("https://openrouter.ai/api/v1/chat/completions")
	if err != nil {
		return "", 0, err
	}

	text := gjson.Get(resp.String(), "choices.0.message.content").String()
	if text == "" {
		return "", 0, fmt.Errorf("no response from LLM")
	}

	score := int(gjson.Get(text, "score").Int())
	return text, score, nil
}
