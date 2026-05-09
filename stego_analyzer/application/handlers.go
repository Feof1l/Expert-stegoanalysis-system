package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"main/database"
	"main/stego"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type FastAPIRequest struct {
	Features []float64 `json:"features"`
}
type contextKey string

const (
	userIDKey contextKey = "user_id"
	emailKey  contextKey = "email"
)

func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}
	app.logger.Debugf("Health check: IP=%s", clientIP)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func (app *Application) registerHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.logger.Infof("Registration attempt failed: invalid request body from IP %s", clientIP)
		app.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app.logger.Infof("Registration attempt: email=%s, name=%s, IP=%s", req.Email, req.Name, clientIP)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		app.logger.Infof("Registration failed: missing required fields for email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	if !emailRegex.MatchString(req.Email) {
		app.logger.Infof("Registration failed: invalid email format for email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	if len(req.Name) < 2 {
		app.logger.Infof("Registration failed: name too short for email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusBadRequest, "name must be at least 2 characters long")
		return
	}

	if len(req.Password) < 8 {
		app.logger.Infof("Registration failed: password too short for email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusBadRequest, "password must be at least 8 characters long")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := app.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		app.logger.Infof("Registration failed: email already exists email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusConflict, "email already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		app.logger.Errorf("Password hash error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	user := database.User{
		Email:          req.Email,
		Name:           req.Name,
		HashedPassword: string(hashedPassword),
	}
	err = app.userRepo.Create(ctx, user)

	if err != nil {
		app.logger.Errorf("Registration DB error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	createdUser, err := app.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		app.logger.Errorf("Failed to fetch created user: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	token, err := app.generateJWT(createdUser.ID, createdUser.Email)
	if err != nil {
		app.logger.Errorf("Token generation error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	app.logger.Infof("Registration successful: user_id=%d, email=%s, name=%s, IP=%s", createdUser.ID, createdUser.Email, createdUser.Name, clientIP)

	app.respondWithJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    createdUser.ID,
			"email": createdUser.Email,
			"name":  createdUser.Name,
		},
	})
}

func (app *Application) loginHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.logger.Infof("Login attempt failed: invalid request body from IP %s", clientIP)
		app.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app.logger.Infof("Login attempt: email=%s, IP=%s", req.Email, clientIP)

	if req.Email == "" || req.Password == "" {
		app.logger.Infof("Login failed: missing required fields for email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := app.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		app.logger.Infof("Login failed: user not found email=%s, IP=%s", req.Email, clientIP)
		app.respondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		app.logger.Infof("Login failed: invalid password for user_id=%d, email=%s, IP=%s", user.ID, req.Email, clientIP)
		app.respondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := app.generateJWT(user.ID, user.Email)
	if err != nil {
		app.logger.Errorf("Token generation error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	app.logger.Infof("Login successful: user_id=%d, email=%s, name=%s, IP=%s", user.ID, user.Email, user.Name, clientIP)

	app.respondWithJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func (app *Application) profileHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	userID, ok1 := r.Context().Value(userIDKey).(int64)
	email, ok2 := r.Context().Value(emailKey).(string)
	if !ok1 || !ok2 {
		app.logger.Warnf("Profile request failed: missing user context, IP=%s", clientIP)
		app.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	app.logger.Infof("Profile request: user_id=%d, email=%s, IP=%s", userID, email, clientIP)

	app.respondWithJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"email":   email,
	})
}

func (app *Application) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.respondWithError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := r.Context().Value(userIDKey).(int64)
	if !ok {
		app.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	app.logger.Infof("Analyze request: user_id=%d, IP=%s", userID, clientIP)

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		app.logger.Errorf("Parse multipart error: %v", err)
		app.respondWithError(w, http.StatusBadRequest, "invalid file")
		return
	}

	file, handler, err := r.FormFile("image")
	if err != nil {
		app.logger.Infof("No image file from user_id=%d: %v", userID, err)
		app.respondWithError(w, http.StatusBadRequest, "image file required")
		return
	}
	defer file.Close()

	filename := filepath.Base(handler.Filename)
	tempPath := fmt.Sprintf("/tmp/%s_%d", filename, userID)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		app.logger.Errorf("Read file error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "file read error")
		return
	}

	if err := os.WriteFile(tempPath, fileBytes, 0644); err != nil {
		app.logger.Errorf("Write temp file error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "file save error")
		return
	}
	defer os.Remove(tempPath)

	pixels, err := stego.LoadPixels(tempPath)
	if err != nil {
		app.logger.Errorf("LoadPixels error for user_id=%d: %v", userID, err)
		app.respondWithError(w, http.StatusInternalServerError, "image processing failed")
		return
	}

	rawFeatures := stego.Extract(pixels)
	var featureSlice []float64
	featureSlice = []float64{
		rawFeatures.LSBTransitions,
		rawFeatures.BitRun00,
		rawFeatures.BitRun01,
		rawFeatures.BitRun10,
		rawFeatures.BitRun11,
		rawFeatures.NeighborDiff,
		rawFeatures.ChiSquare,
		rawFeatures.EntropyLSB,
		rawFeatures.R,
		rawFeatures.S,
		rawFeatures.Rm,
		rawFeatures.Sm,
		rawFeatures.RmR,
		rawFeatures.SmS,
	}

	fastReq := FastAPIRequest{Features: featureSlice}
	featuresJSON, err := json.Marshal(fastReq)
	if err != nil {
		app.logger.Errorf("JSON marshal error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "features encoding failed")
		return
	}
	app.logger.Infof("→ FastAPI request: %s", string(featuresJSON))

	resp, err := http.Post("http://stego-model:8000/predict",
		"application/json",
		bytes.NewBuffer(featuresJSON))
	if err != nil {
		app.logger.Errorf("HTTP POST error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "model service unavailable")
		return
	}
	defer resp.Body.Close()

	app.logger.Infof("FastAPI response status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		app.logger.Errorf("FastAPI error %d: %s", resp.StatusCode, string(body))
		app.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("model error: %d", resp.StatusCode))
		return
	}

	if resp.StatusCode != http.StatusOK {
		app.logger.Errorf("FastAPI bad response: %d", resp.StatusCode)
		app.respondWithError(w, http.StatusInternalServerError, "model prediction failed")
		return
	}

	var fastResp struct {
		StegoProb float64 `json:"stego_prob"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fastResp); err != nil {
		app.logger.Errorf("FastAPI decode error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "model response error")
		return
	}

	prob := fastResp.StegoProb
	result := "likely clean"
	if prob > 0.5 {
		result = "likely contains hidden container"
	}

	app.logger.Infof("Analysis complete: user_id=%d, filename=%s, stego_prob=%.4f, result=%s",
		userID, filename, prob, result)

	featuresWithAnalysis := stego.GetFeaturesWithAnalysisSlice(featureSlice)

	response := stego.AnalyzeResponse{
		StegoProb: prob,
		Result:    result,
		Features:  featuresWithAnalysis,
	}
	app.respondWithJSON(w, http.StatusOK, response)
}

func (app *Application) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	userID, ok := r.Context().Value(userIDKey).(int64)
	email, ok2 := r.Context().Value(emailKey).(string)
	if !ok || !ok2 {
		app.logger.Warnf("Change password request failed: missing user context, IP=%s", clientIP)
		app.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	app.logger.Infof("Change password request: user_id=%d, email=%s, IP=%s", userID, email, clientIP)

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		app.respondWithError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	if len(req.NewPassword) < 8 {
		app.respondWithError(w, http.StatusBadRequest, "new password must be at least 8 characters long")
		return
	}

	if req.OldPassword == req.NewPassword {
		app.respondWithError(w, http.StatusBadRequest, "new password must be different from old password")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := app.userRepo.GetByID(ctx, userID)
	if err != nil {
		app.logger.Errorf("User fetch error: %v", err)
		app.respondWithError(w, http.StatusUnauthorized, "user not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.OldPassword)); err != nil {
		app.respondWithError(w, http.StatusUnauthorized, "invalid old password")
		return
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		app.logger.Errorf("New password hash error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	user.HashedPassword = string(hashedNewPassword)

	err = app.userRepo.UpdatePassword(ctx, user)
	if err != nil {
		app.logger.Errorf("Password update error: %v", err)
		app.respondWithError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	app.logger.Infof("Password changed successfully: user_id=%d, email=%s, IP=%s", userID, email, clientIP)

	app.respondWithJSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

func (app *Application) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.logger.Warnf("Auth failed: missing authorization header, path=%s, IP=%s", r.URL.Path, clientIP)
			app.respondWithError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return app.JWTSecret, nil
		})

		if err != nil || !token.Valid {
			app.logger.Warnf("Auth failed: invalid token, path=%s, IP=%s, error=%v", r.URL.Path, clientIP, err)
			app.respondWithError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userIDFloat, ok := claims["user_id"].(float64)
			if !ok {
				app.logger.Warnf("Auth failed: invalid user_id in token, path=%s, IP=%s", r.URL.Path, clientIP)
				app.respondWithError(w, http.StatusUnauthorized, "invalid user_id in token")
				return
			}
			userID := int64(userIDFloat)

			email, ok := claims["email"].(string)
			if !ok {
				app.logger.Warnf("Auth failed: invalid email in token, path=%s, IP=%s", r.URL.Path, clientIP)
				app.respondWithError(w, http.StatusUnauthorized, "invalid email in token")
				return
			}

			app.logger.Debugf("Auth successful: user_id=%d, email=%s, path=%s, IP=%s", userID, email, r.URL.Path, clientIP)

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, emailKey, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			app.logger.Warnf("Auth failed: invalid token claims, path=%s, IP=%s", r.URL.Path, clientIP)
			app.respondWithError(w, http.StatusUnauthorized, "invalid token claims")
		}
	})
}

func (app *Application) respondWithError(w http.ResponseWriter, code int, message string) {
	app.respondWithJSON(w, code, ErrorResponse{Error: message})
}

func (app *Application) respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		app.logger.Errorf("Failed to encode JSON response: %v", err)
	}
}
