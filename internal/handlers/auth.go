package handlers

import (
	"encoding/json"
	"net/http"
	// "strconv"

	"github.com/unadkatdinky/devpulse/internal/models"
	"github.com/unadkatdinky/devpulse/internal/repository"
	"github.com/unadkatdinky/devpulse/pkg/utils"
)

// AuthHandler holds everything the auth endpoints need to work
// Dependencies are passed in — not grabbed from global variables
type AuthHandler struct {
	UserRepo  *repository.UserRepository
	JWTSecret string
	JWTExpiry int
}

func NewAuthHandler(
	repo *repository.UserRepository,
	secret string,
	expiry int,
) *AuthHandler {
	return &AuthHandler{
		UserRepo:  repo,
		JWTSecret: secret,
		JWTExpiry: expiry,
	}
}

// ── Request bodies ────────────────────────────────────────────────────────────

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ── Response body ─────────────────────────────────────────────────────────────

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// ── Register ──────────────────────────────────────────────────────────────────

// Register handles POST /auth/register
// Creates a new user account and returns a JWT token
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse the JSON body into our registerRequest struct
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate — make sure required fields are present
	if req.Name == "" || req.Email == "" || req.Password == "" {
		utils.JSONError(w, http.StatusBadRequest, "name, email and password are required")
		return
	}

	if len(req.Password) < 8 {
		utils.JSONError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Check if this email is already registered
	existing, err := h.UserRepo.FindByEmail(req.Email)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "database error")
		return
	}
	if existing != nil {
		// 409 Conflict — resource already exists
		utils.JSONError(w, http.StatusConflict, "email already registered")
		return
	}

	// Hash the password before storing
	// req.Password is plain text — we must never save this directly
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	// Build the user and save to database
	user := models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
	}

	if err := h.UserRepo.Create(&user); err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Generate JWT — user is registered and immediately logged in
	token, err := utils.GenerateToken(
		user.ID,
		user.Email,
		h.JWTSecret,
		h.JWTExpiry,
	)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	utils.JSONSuccess(w, http.StatusCreated, authResponse{
		Token: token,
		User:  user,
		// user.PasswordHash is excluded from JSON because of json:"-" on the model
	})
}

// ── Login ─────────────────────────────────────────────────────────────────────

// Login handles POST /auth/login
// Checks credentials and returns a JWT token
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.JSONError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Look up the user
	user, err := h.UserRepo.FindByEmail(req.Email)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Check user exists AND password matches in a single condition
	// Critical security rule: never say "email not found" or "wrong password" separately
	// Always say "invalid email or password"
	// If you reveal which one failed, attackers use your login
	// endpoint to discover which emails are registered in your system
	if user == nil || !utils.CheckPassword(req.Password, user.PasswordHash) {
		utils.JSONError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := utils.GenerateToken(
		user.ID,
		user.Email,
		h.JWTSecret,
		h.JWTExpiry,
	)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	utils.JSONSuccess(w, http.StatusOK, authResponse{
		Token: token,
		User:  *user,
	})
}