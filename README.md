# ExecBrief Backend

The backend service for ExecBrief, an AI-powered meeting intelligence platform that transforms meeting recordings into actionable insights.

## Overview

This Go-based REST API handles:
- User authentication and authorization
- Meeting recording upload and processing
- AI-powered transcription and analysis
- Action item and decision extraction
- Meeting insights and analytics
- Subscription and credit management

## Tech Stack

- **Language**: Go 1.22+
- **Framework**: Gin Web Framework
- **Database**: MongoDB
- **AI Service**: Google Gemini AI
- **Authentication**: JWT-based sessions
- **Storage**: Local file system (configurable for cloud)
- **Validation**: Go-playground validator

## Project Structure

```
backend/
├── cmd/                  # Application entry points
│   └── api/              # Main API server
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── db/               # Database connectivity
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # Custom middleware
│   ├── models/           # Data models and schemas
│   ├── repository/       # Data access layer
│   ├── services/         # Business logic
│   └── ai/               # AI service integrations
└── go.mod                # Go module definition
```

## Features

### Authentication
- User registration with email verification
- Secure login/logout with JWT sessions
- Password hashing with bcrypt
- Role-based access control (free/team plans)
- Session management

### Meeting Processing
- Audio/video file upload (MP4, MP3, WAV, M4A)
- Secure file storage with virus scanning
- Asynchronous processing queue
- Progress tracking for long-running tasks

### AI Capabilities
- Speech-to-text transcription using Gemini
- Meeting summarization and key point extraction
- Action item identification with assignees
- Decision detection and tracking
- Sentiment analysis and engagement metrics
- Topic modeling and keyword extraction

### Data Management
- CRUD operations for meetings, users, action items
- Relationship mapping between entities
- Efficient querying with pagination and filtering
- Data validation and sanitization
- Soft deletion for data recovery

### Subscription & Billing
- Credit-based system for free tier
- Unlimited usage for team plan
- Usage tracking and analytics
- Plan management and upgrades
- Credit expiration policies

## API Documentation

### Base URL
```
/api
```

### Authentication
Most endpoints require authentication via JWT token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

### Endpoints

#### Auth
- `POST /api/auth/signup` - Register new user
- `POST /api/auth/login` - Authenticate user
- `GET /api/auth/me` - Get current user profile
- `PATCH /api/auth/me` - Update user profile
- `PATCH /api/auth/plan` - Update subscription plan
- `POST /api/auth/logout` - Logout user

#### Meetings
- `POST /api/meetings/upload` - Upload meeting recording
- `GET /api/meetings` - List user's meetings
- `GET /api/meetings/{id}` - Get meeting details
- `POST /api/meetings/{id}/analyze` - Trigger AI analysis
- `DELETE /api/meetings/{id}` - Delete meeting

#### Action Items
- `PUT /api/action-items/{id}` - Update action item status

#### Health
- `GET /health` - Service health check

### Request/Response Format
All API communication uses JSON:
- Requests: `Content-Type: application/json`
- Responses: Standard HTTP status codes with JSON bodies

### Error Responses
```json
{
  "error": "Error description",
  "code": "ERROR_CODE"
}
```

## Setup and Installation

### Prerequisites
- Go 1.22 or later
- MongoDB 4.4 or later
- Google Gemini AI API key
- FFmpeg (for audio processing)

### Environment Variables
Create a `.env` file in the backend root:

```env
# Server Configuration
PORT=8080
APP_ENV=development
FRONTEND_ORIGIN=http://localhost:3000

# MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DB=execbrief

# Google Gemini AI
GOOGLE_AI_API_KEY=your_api_key_here

# File Upload
MAX_UPLOAD_MB=100
UPLOAD_DIR=./uploads

# Security
JWT_SECRET=your_super_secret_jwt_key_change_in_production
SESSION_TIMEOUT_HOURS=24

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW_MINUTES=60
```

### Installation Steps
1. Clone the repository
2. Navigate to backend directory
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Set up environment variables (see above)
5. Ensure MongoDB is running
6. Start the server:
   ```bash
   go run cmd/api/main.go
   ```

### Development
- Run tests: `go test ./...`
- Lint code: `golangci-lint run`
- Build binary: `go build -o execbrief-api cmd/api/main.go`
- Docker build: `docker build -t execbrief-backend .`

## Database Schema

### Users Collection
```json
{
  "_id": ObjectId,
  "email": string (unique),
  "password": string (hashed),
  "name": string,
  "plan": enum["free", "team"],
  "credits": integer,
  "maxCredits": integer,
  "createdAt": timestamp,
  "updatedAt": timestamp
}
```

### Meetings Collection
```json
{
  "_id": ObjectId,
  "userId": ObjectId (ref: users),
  "title": string,
  "filename": string,
  "fileSize": integer,
  "duration": integer (seconds),
  "status": enum["uploaded", "processing", "completed", "failed"],
  "transcript": string,
  "summary": string,
  "createdAt": timestamp,
  "updatedAt": timestamp
}
```

### Action Items Collection
```json
{
  "_id": ObjectId,
  "meetingId": ObjectId (ref: meetings),
  "description": string,
  "assignee": string,
  "dueDate": timestamp,
  "status": enum["pending", "in_progress", "completed"],
  "createdAt": timestamp,
  "updatedAt": timestamp
}
```

### Decisions Collection
```json
{
  "_id": ObjectId,
  "meetingId": ObjectId (ref: meetings),
  "description": string,
  "context": string,
  "createdAt": timestamp,
  "updatedAt": timestamp
}
```

### Sessions Collection
```json
{
  "_id": ObjectId,
  "userId": ObjectId (ref: users),
  "token": string (hashed),
  "expiresAt": timestamp,
  "createdAt": timestamp
}
```

## AI Processing Pipeline

1. **File Upload**: User uploads meeting recording
2. **Validation**: File type, size, and virus scan
3. **Storage**: File saved to uploads directory
4. **Transcription**: Audio converted to text using Gemini
5. **Analysis**: Text processed for:
   - Summary extraction
   - Action item identification
   - Decision detection
   - Sentiment analysis
   - Topic modeling
6. **Storage**: Results saved to database
7. **Notification**: User notified of completion

## Security Considerations

- **Authentication**: JWT tokens with 24-hour expiration
- **Authorization**: Role-based access control
- **Data Protection**: 
  - Passwords bcrypt hashed
  - File uploads validated and scanned
  - SQL injection prevention (MongoDB queries)
  - XSS prevention through output encoding
- **Rate Limiting**: Per-IP request limiting
- **CORS**: Configured for frontend origin only
- **Headers**: Security headers implemented (CSP, HSTS, etc.)

## Performance Optimizations

- **Database Indexing**: Strategic indexes on frequently queried fields
- **Connection Pooling**: MongoDB connection reuse
- **Caching**: In-memory caching for frequent lookups
- **Async Processing**: Long-running tasks handled asynchronously
- **File Streaming**: Efficient file upload/download handling
- **Compression**: Gzip compression for API responses

## Testing

### Unit Tests
Run all unit tests:
```bash
go test ./...
```

### Integration Tests
Integration tests require MongoDB instance:
```bash
go test -tags=integration ./...
```

### Test Coverage
Generate coverage report:
```bash
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out
```

## Deployment

### Docker
Build and run with Docker:
```bash
docker build -t execbrief-backend .
docker run -p 8080:8080 --env-file .env execbrief-backend
```

### Kubernetes
See `k8s/` directory for deployment manifests.

### Environment Specific Configurations
- **Development**: Hot reload, verbose logging
- **Staging**: Mirror of production with test data
- **Production**: Optimized settings, monitoring, logging

## Monitoring and Logging

### Logging
- Structured JSON logging
- Log levels: DEBUG, INFO, WARN, ERROR
- Request tracing with unique IDs
- Error tracking with stack traces

### Metrics
- Request latency histograms
- Error rates and counts
- Database query performance
- AI processing times
- Memory and CPU utilization

### Health Checks
- `/health` endpoint returns service status
- Dependency checks (database, external APIs)
- Resource utilization warnings

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

Please follow the Go code review checklist:
- [ ] Code follows Go idioms and conventions
- [ ] Proper error handling
- [ ] Adequate test coverage
- [ ] Documentation for public functions
- [ ] No unused imports or variables
- [ ] Security best practices followed

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contact

For questions and support, please open an issue in the repository or contact the development team.

---
*ExecBrief - Turning meetings into actionable intelligence*