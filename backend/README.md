# RojgarBridge Backend Engine (Golang)

RojgarBridge is a high-performance backend built in **Golang** to automatically scrape, parse, and review latest Indian government job notifications (Sarkari Naukri).

---

## ⚡ The 5-Step Architecture

1. **Scrapers (`internal/scrapers/`)**:
   - Ultra-fast concurrent scrapers (`net/http` + `goquery`).
   - **100% Direct Official Government Portals ONLY** (No 3rd-party aggregators):
     - **SSC** (`ssc.gov.in`) - Staff Selection Commission
     - **UPSC** (`upsc.gov.in`) - Union Public Service Commission
     - **IBPS** (`ibps.in`) - Institute of Banking Personnel Selection
     - **BPSC** (`bpsc.bihar.gov.in`) - Bihar Public Service Commission
     - **DSSSB** (`dsssb.delhi.gov.in`) - Delhi Subordinate Services Selection Board
     - **NTA** (`nta.ac.in`) - National Testing Agency
     - **UPPSC** (`uppsc.up.nic.in`) - Uttar Pradesh Public Service Commission
     - **ISRO** (`isro.gov.in`) - Indian Space Research Organisation
   - Generates SHA-256 deduplication hash (`portal_id + title + url`) so duplicate notices are never processed twice.
2. **AI Extractor (`internal/extractor/`)**:
   - Uses **Google Gemini 1.5 Flash** with strict JSON schemas to extract structured data: Title, Vacancies, Age Limit, Qualification, Fees, Dates, and Official PDF links.
   - Built-in heuristic rule-based fallback when offline or if API key is not configured.
   - Strict domain whitelist validation (only `.gov.in`, `.nic.in`, `.ac.in`, `.edu.in`, `ibps.in` allowed).
3. **Database & Draft Storage (`internal/db/`)**:
   - Embedded **SQLite with WAL mode**, thread-safe write locks (`sync.Mutex`), and connection busy timeouts.
   - Dynamic multi-condition filtering (`status`, `notice_type`, `category`, `state`).
4. **Reliability & Review Layer (`internal/api/`)**:
   - Admin can view, edit, and verify parsed details against the source notice.
   - One-click `POST /api/jobs/{id}/publish` or `POST /api/jobs/bulk-publish`.
   - Automated **Telegram Channel Alerts** upon publishing.
5. **Live Public Feed & SEO**:
   - `GET /api/jobs/published` serves verified, clean job announcements for the frontend.
   - `GET /api/jobs/search?q=...` full-text search across titles, organizations, and categories.
   - `GET /api/jobs/closing-soon?days=7` notices with approaching deadlines.
   - `GET /api/jobs/analytics` real-time aggregate statistics.
   - `GET /feed.xml` / `/rss.xml` standard RSS 2.0 feed for aggregators & search crawlers.
   - `GET /sitemap.xml` XML sitemap for search engines.

---

## 🚀 API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Service health status |
| `GET` | `/api/jobs` | Filterable job listing (`?status=draft|published`, `?category=...`, `?state=...`, `?type=...`) |
| `GET` | `/api/jobs/search` | Multi-field search (`?q=keyword&status=...`) |
| `GET` | `/api/jobs/closing-soon`| Urgent jobs with approaching deadlines (`?days=7`) |
| `GET` | `/api/jobs/analytics` | Real-time aggregate portal metrics and state breakdowns |
| `GET` | `/api/jobs/drafts` | List all unverified drafts |
| `GET` | `/api/jobs/published` | List all verified live jobs |
| `GET` | `/api/jobs/{id}` | Get full details of a specific job |
| `PUT` | `/api/jobs/{id}` | Update job details prior to publishing |
| `POST` | `/api/jobs/{id}/publish` | Approve draft, publish to live site, and alert Telegram |
| `POST` | `/api/jobs/bulk-publish` | Bulk publish drafts by ID array or all pending |
| `POST` | `/api/jobs/{id}/reject` | Reject/archive an invalid draft |
| `POST` | `/api/pipeline/run` | Trigger scraping & AI extraction on-demand |
| `POST` | `/api/pipeline/re-extract` | Re-run heuristic extraction across existing drafts |
| `GET` | `/feed.xml` | Standard RSS 2.0 XML feed |
| `GET` | `/rss.xml` | Alias for RSS 2.0 XML feed |
| `GET` | `/sitemap.xml` | XML sitemap for SEO crawlers |

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
export TELEGRAM_BOT_TOKEN="your-telegram-bot-token"
export TELEGRAM_CHAT_ID="@your_channel_or_chat_id"
```

### 3. Start Server
```bash
./server
```

The server automatically runs an initial scraping pass on boot, and runs background passes every 6 hours.

