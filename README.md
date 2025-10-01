# CV & Project Report Evaluator

A mini AI-driven system to evaluate CVs and project reports asynchronously. Built with **Go**, **Fiber**, **PostgreSQL**, and **Gemini LLM**, this project demonstrates how to integrate PDF OCR, vector databases, and LLM evaluation into a clean, scalable architecture.

---

## Features

- **Async Evaluation**: Users can upload CVs and project reports without waiting for AI processing.
- **OCR PDF Extraction**: Extract text from PDFs, including scanned documents, using Tesseract OCR.
- **Vector Database**: Job descriptions are embedded and stored in PostgreSQL with `pgvector` for RAG retrieval.
- **LLM Integration**: Uses Gemini LLM for evaluating CVs and project reports with structured JSON output.
- **Resilient Design**: Retries, backoff, circuit breakers, and low-temperature LLM calls to ensure consistent results.
- **Clean Architecture**: Organized into `usecase`, `repository`, `service`, and `handler` layers.

---

## Tech Stack

- **Language & Framework**: Go + Fiber
- **Database**: PostgreSQL + `pgvector`
- **PDF Extraction**: Tesseract OCR + `go-fitz`
- **LLM**: Gemini (Google) for embeddings and evaluation
- **Env Management**: godotenv

---

## System Design

### Endpoints

1. `POST /evaluate` – Upload CV and project report. Returns a `job_id`.
2. `GET /result/{id}` – Fetch evaluation result using the `job_id`.

### Database Schema

**evaluation_tasks**  

| Field               | Type         | Description |
|--------------------|-------------|-------------|
| id                  | UUID        | Primary Key |
| cv                  | Text        | Extracted CV content |
| report              | Text        | Extracted project report content |
| status              | Varchar(50) | `processing`, `done`, `failed` |
| cv_match_rate       | Float       | CV match score |
| cv_feedback         | Text        | CV feedback text |
| project_score       | Float       | Project score |
| project_feedback    | Text        | Project report feedback |
| overall_summary     | Text        | Summary of evaluation |
| breakdown           | JSONB       | Detailed breakdown scores |
| result              | JSONB       | Full JSON evaluation |
| created_at          | Timestamp   | Created timestamp |
| updated_at          | Timestamp   | Updated timestamp |

**jobs**  

| Field     | Type       | Description |
|-----------|-----------|-------------|
| id        | UUID      | Primary Key |
| title     | Text      | Job title |
| content   | Text      | Job description |
| embedding | Vector    | Vector embedding for RAG |
| created_at| Timestamp | Timestamp |
| updated_at| Timestamp | Timestamp |

---

## Usage

### Setup

1. Clone the repository:

```bash
git clone https://github.com/yourusername/cv-analyzer.git
cd cv-analyzer
```

2. Copy .env.example to .env and set your environment variables
3. Run Postgres with pgvector extension (Docker recommended):
```bash
docker run --name cv-analyzer-postgres -e POSTGRES_PASSWORD=pass -e POSTGRES_USER=user -p 5433:5432 -d postgres:15
```
4. Install dependencies and run the server:
```bash
go mod tidy
go run cmd/server/main.go
```

---

## Endpoints

1. POST /evaluate
```bash
curl -X POST http://localhost:8080/evaluate \
-F "cv=@/path/to/cv.pdf" \
-F "project_report=@/path/to/report.pdf"
```
2. GET /result/{id}
```bash
curl http://localhost:8080/result/<id>
```

---

## How It Works

1. PDF Extraction: Extracts text from CV and project report using Tesseract OCR.
2. Embedding Jobs: All job descriptions are converted into embeddings and stored in Postgres.
3. RAG Retrieval: Relevant job info is retrieved based on embedding similarity.
4. LLM Evaluation: Gemini evaluates the CV and project report, returning JSON with scores, feedback, and breakdowns.
5. Async Handling: Evaluation runs in a goroutine; /evaluate responds immediately with a id.

---

## Notes & Limitations

1. Gemini API free tier is rate-limited → endpoints use 1 request per 4 seconds.
2. OCR accuracy depends on PDF quality.
3. Only supports English PDFs for OCR.

---

## Future Improvements

1. Add CI/CD integration.
2. Support more languages for OCR.
3. Improve error handling and monitoring for long-running tasks.
4. Enhance RAG retrieval for better context scoring.
