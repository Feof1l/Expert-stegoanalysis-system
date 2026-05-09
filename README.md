# StegoGuard - Intelligent Expert System for Steganalysis

<div align="center">

![Architecture](https://img.shields.io/badge/Architecture-Microservices-green)
![Go](https://img.shields.io/badge/Go-1.25-blue)
![Python](https://img.shields.io/badge/Python-FastAPI-blue)
![License](https://img.shields.io/badge/License-MIT-yellow)

**An intelligent steganalysis system that detects hidden content in digital images using machine learning**

</div>

---

## Overview

StegoGuard is an expert system for detecting steganographic content in digital images containers. The system analyzes images using 14 statistical features and predicts the probability of hidden data using a trained machine learning model.

### What is Steganalysis?

Steganalysis is the science of detecting hidden information within digital media. This project implements classic statistical methods to identify whether an image contains steganographic content (hidden data embedded in the image pixels).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                   │
│      ┌─────────────────────────────────────────────────────────────────┐    │
│      │                        Frontend (nginx)                         │    │
│      │              http://localhost:3000 (static HTML/JS)             │    │
│      └─────────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ HTTP
                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                             API GATEWAY LAYER                           │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                  Backend API (Go, port 8080)                    │   │
│   │                                                                 │   │
│   │   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐    │   │
│   │   │ /health  │  │/register │  │ /login   │  │ /api/analyze │    │   │
│   │   └──────────┘  └──────────┘  └──────────┘  └──────────────┘    │   │
│   │                    │                                    │       │   │
│   │                    ▼                                    ▼       │   │
│   │ ┌─────────────────────────────────────────────────────────────┐ │   │
│   │ │              Authentication (JWT + BCrypt)                  │ │   │
│   │ └─────────────────────────────────────────────────────────────┘ │   │
│   └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                    ┌────────────────┴────────────────┐
                    │                │                │                
                    ▼                ▼                ▼                
    ┌──────────────────┐    ┌──────────────┐      ┌──────────────────┐
    │  POSTGRESQL DB   │    │  ML SERVICE  │      │  FEATURE         │
    │     (port 5432)  │    │ (port 8000)  │      │  EXTRACTION      │
    │                  │    │              │      │                  │
    │  - users table   │    │  - FastAPI   │      │  - LSB Analysis  │
    │  - user_id       │    │  - scikit-   │      │  - Chi-Square    │
    │  - email         │    │    learn     │      │  - RS Analysis   │
    │  - hashed_pw     │    │    model     │      │  - Entropy       │
    │  - name          │    │              │      │                  │
    └──────────────────┘    └──────────────┘      └──────────────────┘
```

---

## How It Works (Business Logic)

### 1. User Authentication Flow

```
User Registration                    User Login
       │                                 │
       ▼                                 ▼
┌──────────────────┐           ┌──────────────────┐
│  Validate input  │           │  Find user by    │
│  (email, pass,   │           │  email           │
│   name)          │           └────────┬─────────┘
└────────┬─────────┘                    │
         │                              ▼
         ▼                     ┌─────────────────┐
┌──────────────────┐           │  Compare bcrypt │
│  Check email     │           │  hash with      │
│  uniqueness      │           │  password       │
└────────┬─────────┘           └────────┬────────┘
         │                              │
         ▼                              ▼
┌──────────────────┐           ┌──────────────────┐
│  Hash password   │           │  Generate JWT    │
│  (bcrypt)        │           │  (72h expiry)    │
└────────┬─────────┘           └────────┬─────────┘
         │                              │
         ▼                              ▼
┌──────────────────┐           ┌──────────────────┐
│  Insert to DB    │           │  Return token    │
│  Return user     │           │  + user data     │
└──────────────────┘           └──────────────────┘
```

### 2. Image Analysis Flow

```
User uploads image                    Extract pixel data
       │                                 │
       ▼                                 ▼
┌──────────────────┐           ┌──────────────────┐
│  Validate JWT    │           │  Decode image    │
│  token           │           │  (PNG/JPEG)      │
└────────┬─────────┘           └────────┬─────────┘
         │                              │
         ▼                              ▼
┌──────────────────┐           ┌──────────────────┐
│  Get user_id     │           │  Extract 14      │
│  from context    │           │  statistical     │
└────────┬─────────┘           │  features        │
         │                     └────────┬─────────┘
         │                               │
         ▼                               ▼
         │                    ┌─────────────────────┐
         │                    │  1. LSB Transition  │
         │                    │  2. Bit Run (00-11) │
         │                    │  3. Neighbor Diff   │
         │                    │  4. Chi-Square      │
         │                    │  5. LSB Entropy     │
         │                    │  6. RS Features     │
         │                    │     (R, S, Rm, Sm)  │
         │                    │  7. Rm-R, Sm-S      │
         │                    └──────────┬──────────┘
         │                               │
         ▼                               ▼
┌────────────────────────────────────────────────────────────────┐
│                    Send features to ML Model                   │
└─────────────────────────────────┬──────────────────────────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│                    ML Model Prediction                           │
│                                                                  │
│   Input: [14 features] → scikit-learn → [stego_probability]      │
│   Output: {"stego_prob": 0.7234}                                 │
└─────────────────────────────────┬────────────────────────────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Return Result to Client                       │
│                                                                  │
│   {                                                              │
│     "stego_prob": 0.72,     // ML probability                    │
│     "result": "Hidden content detected",                         │
│     "features": [           // Feature analysis                  │
│       {"name": "LSB Transition", "value": 0.52,                  │
│        "min_normal": 0.45, "max_normal": 0.55,                   │
│        "is_anomaly": false},                                     │
│       ...                                                        │
│     ]                                                            │
│   }                                                              │
└──────────────────────────────────────────────────────────────────┘
```

---

## Services

| Service | Technology | Port | Description |
|---------|-----------|------|-------------|
| **Frontend** | HTML/CSS/JS | 3000 | User interface, static files served by nginx |
| **Backend** | Go 1.25 | 8080 | REST API, authentication, image processing |
| **ML Model** | Python + FastAPI | 8000 | Steganalysis prediction service |
| **Database** | PostgreSQL | 5432 | User data storage |

---

## Feature Extraction Algorithms

The system uses 14 statistical features commonly used in steganalysis:

| # | Feature | Description |
|---|---------|-------------|
| 1 | LSB Transition | Frequency of LSB bit changes between adjacent pixels |
| 2-5 | Bit Run (00, 01, 10, 11) | Statistics of LSB pair transitions |
| 6 | Neighbor Diff | LSB difference between adjacent pixels |
| 7 | Chi-Square | Chi-square test for LSB distribution uniformity |
| 8 | Entropy LSB | Shannon entropy of LSB bits |
| 9-10 | R, S | RS analysis: Regular and Singular group ratios |
| 11-12 | Rm, Sm | RS groups after LSB flipping |
| 13-14 | Rm-R, Sm-S | Difference features for detection |

---

## Quick Start

### Prerequisites

- Docker
- Docker Compose

### Running the System

```bash
# Clone and navigate to project
cd stego_analyze

# Create .env file
echo "JWT_SECRET=your-secret-key-here" > .env

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f
```

### Access

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **ML Model API**: http://localhost:8000
- **Health Check**: http://localhost:8080/health

---

## API Endpoints

### Public Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/register` | Register new user |
| POST | `/login` | Login, returns JWT |

### Protected Endpoints (require JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/analyze` | Analyze image for steganography |
| POST | `/api/change-password` | Change user password |

### Request Examples

**Register:**
```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass1234","name":"John"}'
```

**Login:**
```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass1234"}'
```

**Analyze Image:**
```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Authorization: Bearer <token>" \
  -F "image=@image.png"
```

---

## Project Structure

```
expert_steanalysis_system/
├── docker-compose.yml       # Orchestration
├── .env                    # Environment variables
│
├── frontend/               # Web interface
│   ├── index.html         # Main page
│   ├── styles.css         # Styles
│   ├── Dockerfile         # nginx container
│   └── frontend.yml       # Compose config
│
├── stego_analyzer/         # Backend (Go)
│   ├── main.go            # Entry point
│   ├── config/            # Configuration
│   ├── application/       # Handlers & routing
│   ├── database/          # PostgreSQL repository
│   ├── logger/           # Logging
│   ├── stego/            # Feature extraction
│   │   ├── features.go   # 14 feature algorithms
│   │   ├── loader.go    # Image decoding
│   │   └── stego.go      # Response types
│   ├── Dockerfile        # Go container
│   └── backend.yml       # Compose config
│
├── stego_model/           # ML Service (Python)
│   ├── server.py         # FastAPI app
│   ├── model.py          # ML logic
│   ├── stego_model.pkl   # Trained model (not in repo)
│   ├── requirements.txt  # Python deps
│   ├── Dockerfile        # Python container
│   └── fastapi.yml       # Compose config
│
└── db/                    # Database
    ├── Dockerfile        # PostgreSQL
    ├── initdb.sql        # Schema
    └── db.yml           # Compose config
```

---

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.25, gorilla/mux, golang-jwt, sqlx |
| ML Service | Python 3.x, FastAPI, scikit-learn, joblib |
| Frontend | HTML5, CSS3, Vanilla JS |
| Database | PostgreSQL |
| Container | Docker, nginx |

---

## Security Features

- **Password Storage**: BCrypt hashing (cost factor 10)
- **Authentication**: JWT tokens with 72-hour expiry
- **CORS**: Whitelist-based origin control
- **Input Validation**: Email regex, password/name length limits
- **File Upload**: 10MB size limit

---

## License

MIT License

---

