package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fadilmartias/cv-analyzer/internal/model"
	"github.com/fadilmartias/cv-analyzer/internal/repository"
	"github.com/fadilmartias/cv-analyzer/internal/service"
	"github.com/pgvector/pgvector-go"
	"github.com/tidwall/gjson"
)

type EvaluationUsecase struct {
	evaluationRepo *repository.EvaluationRepository
	jobRepo        *repository.JobRepository
	openRouter     service.OpenRouterServiceInterface
	gemini         service.GeminiServiceInterface
}

func NewEvaluationUsecase(evaluationRepo *repository.EvaluationRepository, jobRepo *repository.JobRepository, openRouter service.OpenRouterServiceInterface, gemini service.GeminiServiceInterface) *EvaluationUsecase {
	return &EvaluationUsecase{evaluationRepo: evaluationRepo, jobRepo: jobRepo, openRouter: openRouter, gemini: gemini}
}

func (uc *EvaluationUsecase) Submit(req model.EvaluationTask) (string, error) {
	req.Status = "processing"
	req.Breakdown = "{}"
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	if err := uc.evaluationRepo.CreateTask(&req); err != nil {
		return "", err
	}

	go uc.EvaluateTask(&req)

	return req.ID.String(), nil
}

func (uc *EvaluationUsecase) CreateJobEmbedding() error {
	ctx := context.Background()
	jobs := []model.Job{
		{
			Title: "Product Engineer (Backend)",
			Content: `You'll be building new product features alongside a frontend engineer and product manager using our Agile methodology, as well as addressing issues to ensure our apps are robust and our codebase is clean. As a Product Engineer, you'll write clean, efficient code to enhance our product's codebase in meaningful ways.

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

This is a remote job. You're free to work where you work best: home office, co-working space, coffee shops. To ensure time zone overlap with our current team and maintain well communication, we're only looking for people based in Indonesia.`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Frontend Engineer",
			Content:   "React, Typescript, Tailwind",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "UI/UX Engineer",
			Content:   "Figma, UI, UX",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	for i, job := range jobs {
		result, err := uc.gemini.GenerateEmbedding(ctx, job.Content)
		if err != nil {
			log.Fatal(err)
		}
		emb := pgvector.NewVector(result)
		jobs[i].Embedding = emb
		uc.jobRepo.UpdateJob(&jobs[i])
	}

	return nil
}

func (uc *EvaluationUsecase) EvaluateTask(task *model.EvaluationTask) error {
	ctx := context.Background()

	// 1️⃣ Generate embedding dari CV
	cvEmb, err := uc.gemini.GenerateEmbedding(ctx, task.CV)
	log.Println("CV Embedding:", cvEmb)
	if err != nil {
		task.Status = "failed"
		_ = uc.evaluationRepo.UpdateTask(task)
		return err
	}

	cvVector := pgvector.NewVector(cvEmb)

	// 2️⃣ Ambil job descriptions relevan (RAG)
	jobs, err := uc.jobRepo.SearchJobs(cvVector, 5)
	log.Println("Jobs:", jobs)
	if err != nil {
		task.Status = "failed"
		_ = uc.evaluationRepo.UpdateTask(task)
		return err
	}

	// 3️⃣ Buat prompt dengan RAG context
	jobContext := ""
	for i, j := range jobs {
		jobContext += fmt.Sprintf("Job %d: %s\nRequirements: %s\n\n", i+1, j.Title, j.Content)
	}

	log.Println("Job Context:", jobContext)

	prompt := fmt.Sprintf(`
You are an experienced technical recruiter. Analyze the following CV and Project Report against these job requirements:

%s

Return your answer STRICTLY in JSON format with this schema:
{
	"cv_match_rate": <float with 2 decimal places, range 0-1 based on cv breakdown score that converted to percents and then x20>,
	"cv_feedback": "<feedback about CV>",
	"project_score": <float with 2 decimal places, range 0-10 based on project breakdown score>,
	"project_feedback": "<feedback about Project Report>",
	"overall_summary": "<summary of overall impression, strengths, and areas to improve>",
  "breakdown": {
    "cv": {
	"technical_skills_match": <number 1-5, weight: 40 percents, criteria: backend, databases, APIs, cloud, and AI/LLM exposure>,
	"experience_level": <number 1-5, weight: 25 percents, criteria: years, project complexity>,
	"relevant_achievements": <number 1-5, weight: 20 percents, criteria: impact, scale>,
	"cultural_fit": <number 1-5, weight: 15 percents, criteria: communication, learning attitude>,
	},
    "project_report": {
      "correctness": <number 1-5, weight: 30 percents, criteria: prompt design, chaining, RAG, handling errors>,
      "code_quality": <number 1-5, weight: 25 percents, criteria: clean, modular, testable>,
      "resilience": <number 1-5, weight: 20 percents, criteria: handles failures, retries>,
      "documentation": <number 1-5, weight: 15 percents, criteria: clear README, explanation of trade-offs>,
      "creativity_or_bonus": <number 1-5, weight: 10 percents, criteria: optional improvements like authentication, deployment, dashboards, etc.>
    }
  }
}

CV:
%s

Report:
%s
`, jobContext, task.CV, task.Report)

	// 4️⃣ Generate evaluation via Gemini
	result, err := uc.gemini.GenerateContent(ctx, "gemini-2.5-flash", prompt)
	log.Println("Result:", result.Text())
	if err != nil {
		task.Status = "failed"
		_ = uc.evaluationRepo.UpdateTask(task)
		return err
	}

	log.Println("Result:", result.Text())

	text := result.Text()
	cvMatchRate := gjson.Get(text, "cv_match_rate").Float()
	cvFeedback := gjson.Get(text, "cv_feedback").String()
	projectScore := gjson.Get(text, "project_score").Float()
	projectFeedback := gjson.Get(text, "project_feedback").String()
	overallSummary := gjson.Get(text, "overall_summary").String()
	breakdown := gjson.Get(text, "breakdown").String()

	// 5️⃣ Update task
	task.CvMatchRate = cvMatchRate
	task.CvFeedback = cvFeedback
	task.ProjectScore = projectScore
	task.ProjectFeedback = projectFeedback
	task.OverallSummary = overallSummary
	task.Breakdown = breakdown
	task.Status = "completed"
	return uc.evaluationRepo.UpdateTask(task)
}

func (uc *EvaluationUsecase) GetResult(id string) (*model.EvaluationTask, error) {
	return uc.evaluationRepo.FindTaskByID(id)
}

func (uc *EvaluationUsecase) Test() (string, error) {
	return uc.gemini.Test()
}
