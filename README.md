# RojgarBridge Backend Engine (Golang)

RojgarBridge is a high-performance backend built in **Golang** to automatically scrape, parse, and review latest Indian government job notifications (Sarkari Naukri).

---

## ⚡ The 5-Step Architecture

1. **Scrapers (`internal/scrapers/`)**:
   - Ultra-fast concurrent scrapers (`net/http` + `goquery`).
   - Monitors official portals (e.g., SSC, UPSC, Employment News, RSS Feeds).
   - Generates SHA-256 deduplication hash (`portal_id + title + url`) so duplicate notices are never processed twice.
2. **AI Extractor (`internal/extractor/`)**:
   - Uses **Google Gemini 1.5 Flash** with strict JSON schemas to extract structured data: Title, Vacancies, Age Limit, Qualification, Fees, Dates, and Official PDF links.
   - Built-in heuristic rule-based fallback when offline or if API key is not configured.
3. **Database & Draft Storage (`internal/db/`)**:
   - Embedded **SQLite with WAL mode** (zero configuration, high throughput).
   - Every scraped item is saved with `status: 'draft'`.
4. **Reliability & Review Layer (`internal/api/`)**:
   - Admin can view, edit, and verify parsed details against the source notice.
   - One-click `POST /api/jobs/:id/publish` changes status to `published`.
5. **Live Public Feed**:
   - `GET /api/jobs/published` serves verified, clean job announcements for the frontend.

---

## 🚀 API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Service health status |
| `GET` | `/api/jobs/drafts` | List all unverified drafts (Filter with `?type=recruitment`, `?type=result`) |
| `GET` | `/api/jobs/published` | List all verified live jobs (Filter with `?type=recruitment`) |
| `GET` | `/api/jobs/{id}` | Get full details of a specific job |
| `PUT` | `/api/jobs/{id}` | Update job details prior to publishing |
| `POST` | `/api/jobs/{id}/publish` | Approve draft and publish to live site |
| `POST` | `/api/jobs/{id}/reject` | Reject/archive an invalid draft |
| `POST` | `/api/pipeline/run` | Trigger scraping & AI extraction on-demand |

---

## 🛠️ How to Run

### 1. Build
```bash
go build -o server .
```

### 2. Configure Environment (Optional)
```bash
export PORT=8080
export DB_PATH=rojgarbridge.db
export GEMINI_API_KEY="your-gemini-api-key"
```

### 3. Start Server
```bash
./server
```

The server automatically runs an initial scraping pass on boot, and runs background passes every 6 hours.
