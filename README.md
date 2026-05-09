# Expert Steganalysis System

## Project Overview

Multi-service steganography detection system that analyzes digital images for hidden content. The system extracts 14 statistical features from images pixels data and uses a machine learning model to predict the probability of steganography.

### Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Frontend   │────▶│    Go API    │────▶│   Python ML  │
│  (static)   │     │  (port 8080) │     │  (port 8000) │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  PostgreSQL │
                    │    (db)     │
                    └─────────────┘
```

### Services

| Service | Directory | Language | Port | Purpose |
|---------|-----------|----------|------|---------|
| Backend | `stego_analyzer/` | Go 1.25 | 8080 | REST API, auth, image processing |
| ML Model | `stego_model/` | Python | 8000 | Feature-based steganalysis prediction |
| Frontend | `frontend/` | HTML/JS | 3000 | User interface |
| Database | `db/` | PostgreSQL | 5432 | User data storage |

---

## Essential Commands

### Running the System

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Rebuild specific service
docker-compose build <service-name>
docker-compose up -d <service-name>
```

### Backend (Go)

```bash
# Development (from stego_analyzer directory)
go run .

# Build binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -trimpath -o backend .

# Run tests
go test ./...
```

### ML Model (Python)

```bash
# From stego_model directory with venv activated
python server.py

# Install dependencies
pip install -r requirements.txt
```

---

## Configuration

### Environment Variables

Create `.env` file in project root:

```bash
# Required
JWT_SECRET=your-secret-key-here

# Database (auto-derived by docker-compose)
DATABASE_URI=postgresql://postgres:postgres@db:5432/postgres

# Optional (have defaults)
LOG_LEVEL=1          # 0=debug, 1=info, 2=warn, 3=error
LOG_DIR=./logs
ADDR=:8080
```

### Config Loading

- Go backend loads config from `.env` via `github.com/joho/godotenv` + `github.com/caarlos0/env/v11`
- Config path in code: `config.Load()` tries to load `../.env` relative to working dir
- **Critical**: JWT_SECRET must be non-empty or app fails to start

---

## Code Patterns & Conventions

### Go Backend

**Package Structure:**
```
stego_analyzer/
├── main.go           # Entry point
├── config/           # Configuration loading
├── application/     # HTTP handlers, routing, auth
├── database/        # DB connection, repository pattern
├── logger/          # Structured logging with rotation
└── stego/           # Feature extraction algorithms
```

**Naming:**
- Lowercase filenames: `database.go`, `handlers.go`
- PascalCase types/functions: `type Application struct`, `func NewApplication()`
- camelCase variables: `db`, `logger`, `userRepo`

**Database:**
- Uses `github.com/jmoiron/sqlx` for queries building
- Repository pattern: `UserRepository` struct with methods
- PostgreSQL with `github.com/lib/pq` driver (imported as `_` for side effect)

**HTTP:**
- Uses `github.com/gorilla/mux` for routing
- Context-based auth: user ID stored in `context.WithValue(r.Context(), userIDKey, userID)`
- CORS middleware with whitelist (hardcoded origins list in `corsMiddleware`)

**Auth:**
- JWT via `github.com/golang-jwt/jwt/v5`
- Password hashing via `golang.org/x/crypto/bcrypt`
- Token includes `user_id` (float64) and `email` claims

### Python ML Model

**Structure:**
```
stego_model/
├── server.py         # FastAPI app, /predict endpoint
├── model.py         # (if exists) ML logic
├── stego_model.pkl  # Trained scikit-learn model (joblib)
└── requirements.txt # Dependencies
```

**Prediction:**
- Input: JSON `{"features": [14 floats]}`
- Output: JSON `{"stego_prob": float}` (probability 0-1)
- Model uses `predict_proba()` returning `[class_0_prob, class_1_prob]`

### Frontend

- Single `index.html` with embedded CSS and JS
- API endpoint hardcoded: `const BACKEND_URL = "http://localhost:8080"`
- Uses Fetch API with JSON for auth, FormData for file upload
- Has artificial delay (line 312-313): `const demoDelay = Math.random() * 4000 + 1000` - this appears to be for demo purposes, actual backend call happens after

---

## API Endpoints

### Public (no auth)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/health` | Health check |
| POST | `/register` | Create account |
| POST | `/login` | Authenticate |

### Protected (requires JWT)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/profile` | Get current user |
| POST | `/api/change-password` | Update password |
| POST | `/api/analyze` | Upload image for steganalysis |

### Request/Response Formats

**Register/Login:**
```json
// Request
{"email": "string", "password": "string", "name": "string"}

// Response
{"token": "jwt-token", "user": {"id": int, "email": "string", "name": "string"}}
```

**Analyze (multipart/form-data):**
```json
// Request: form-data with field "image" (file)

// Response
{"stego_prob": 0.7234, "result": "likely contains hidden container"}
// Threshold: prob > 0.5 = "likely contains hidden container"
```

---

## Feature Extraction (14 features)

Located in `stego_analyzer/stego/features.go`:

| Feature | Description |
|---------|-------------|
| LSBTransitions | LSB bit flip frequency |
| BitRun00-11 | LSB pair transition statistics |
| NeighborDiff | Adjacent pixel LSB difference |
| ChiSquare | Chi-square test on LSB distribution |
| EntropyLSB | Shannon entropy of LSB |
| R, S | RS analysis regular/singular groups |
| Rm, Sm | RS after LSB flip |
| RmR, SmS | Differences |

---

## Important Gotchas

1. **CORS is restrictive**: Only `http://frontend_app:3000` and `http://localhost:3000` allowed. Backend fails silently for other origins.

2. **Docker networking**: Backend calls FastAPI at `http://stego-model:8000` (docker DNS), not `localhost:8000`.

3. **JWT claims**: `user_id` is stored as float64 in JWT, parsed as `float64` then cast to `int64` (line 481-487 in handlers.go).

4. **Password validation**:
   - Min 8 characters
   - Name min 2 characters
   - Email must match regex

5. **File upload limits**: Max 10MB (`10 << 20` in `ParseMultipartForm` call at line 261).

6. **Model file missing**: `stego_model/stego_model.pkl` is not in repo (gitignored) - will fail to load if not volume-mounted or created.

7. **Frontend demo delay**: Artificial 1-5 second delay in `handleAnalyze()` before actual API call - this is likely for UX demo.

8. **Go module name**: Module is `main` (not typical path), imports paths like `main/application`.

9. **PostgreSQL init**: `db/initdb.sql` creates `users` table with auto-increment `user_id`.

10. **Logging**: Uses `gopkg.in/natefinch/lumberjack.v2` for rotation; log directory must be writable.

---

## Development Notes

- No test files visible in the codebase
- No Makefile or build scripts (uses docker-compose)
- Frontend is static HTML - no build step required
- Model training code not included (just pre-trained `.pkl` file)
- Logs written to `./logs` relative to working dir (needs write permission)
