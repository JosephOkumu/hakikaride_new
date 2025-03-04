package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
	"crypto/rand"
	"encoding/base64"
	"golang.org/x/crypto/bcrypt"
	"github.com/dgrijalva/jwt-go"
)

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	UserType string `json:"userType"`
}

type UserData struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Password  string `json:"password"`
	UserType  string `json:"userType"`
}

type Claims struct {
	UserID   int    `json:"userId"`
	UserType string `json:"userType"`
	jwt.StandardClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(userID int, userType string) (string, error) {
	// Create the JWT claims
	claims := &Claims{
		UserID:   userID,
		UserType: userType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with secret key
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GenerateSecurePassword(length int) (string, error) {
	if length < 8 {
		length = 8 // Minimum password length for security
	}

	// Calculate how many bytes we need for the requested length
	// Each byte will be converted to ~1.3 characters in base64
	byteLength := (length * 3) / 4
	if byteLength < 1 {
		byteLength = 1
	}

	// Generate random bytes
	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Convert to base64 string
	password := base64.StdEncoding.EncodeToString(randomBytes)

	// Trim to desired length
	if len(password) > length {
		password = password[:length]
	}

	return password, nil
}

func validateCredentials(creds Credentials) (string, bool) {
	if creds.Email == "" {
		return "Email or phone number is required", false
	}
	if creds.Password == "" {
		return "Password is required", false
	}
	if creds.UserType == "" {
		return "User type is required", false
	}
	if creds.UserType != "parent" && creds.UserType != "driver" && creds.UserType != "admin" {
		return "Invalid user type", false
	}
	return "", true
}

func HandleLogin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid request format",
			})
			return
		}

		// Validate credentials
		if msg, valid := validateCredentials(creds); !valid {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": msg,
			})
			return
		}

		// Get user from database
		var user struct {
			UserID              int
			PasswordHash        string
			UserType            string
			PasswordResetRequired bool
		}

		var err error
		
		// For drivers, we treat the email field as phone number
		if creds.UserType == "driver" {
			err = db.QueryRow(`
				SELECT u.UserID, u.PasswordHash, u.UserType, u.PasswordResetRequired 
				FROM Users u
				JOIN Drivers d ON u.UserID = d.UserID
				WHERE d.PhoneNumber = ? AND u.UserType = ? AND u.IsActive = true`,
				creds.Email, creds.UserType).Scan(&user.UserID, &user.PasswordHash, &user.UserType, &user.PasswordResetRequired)
		} else {
			// For other user types, use email as usual
			err = db.QueryRow(`
				SELECT UserID, PasswordHash, UserType, PasswordResetRequired 
				FROM Users 
				WHERE Email = ? AND UserType = ? AND IsActive = true`,
				creds.Email, creds.UserType).Scan(&user.UserID, &user.PasswordHash, &user.UserType, &user.PasswordResetRequired)
		}

		if err != nil {
			if err == sql.ErrNoRows {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Invalid credentials",
				})
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Check password
		if !CheckPasswordHash(creds.Password, user.PasswordHash) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid credentials",
			})
			return
		}

		// Generate JWT token
		token, err := GenerateToken(user.UserID, user.UserType)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		// Update last login
		_, err = db.Exec("UPDATE Users SET LastLogin = CURRENT_TIMESTAMP WHERE UserID = ?", user.UserID)
		if err != nil {
			// Log error but don't fail the login
			log.Printf("Error updating last login: %v", err)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"token":    token,
			"userType": user.UserType,
			"passwordResetRequired": user.PasswordResetRequired,
		})
	}
}

func validateUserData(userData UserData) (string, bool) {
	if userData.Email == "" {
		return "Email is required", false
	}
	if userData.Password == "" {
		return "Password is required", false
	}
	if len(userData.Password) < 8 {
		return "Password must be at least 8 characters long", false
	}
	if userData.UserType == "" {
		return "User type is required", false
	}
	if userData.UserType != "parent" && userData.UserType != "driver" && userData.UserType != "admin" {
		return "Invalid user type", false
	}
	if userData.FirstName == "" {
		return "First name is required", false
	}
	if userData.LastName == "" {
		return "Last name is required", false
	}
	if userData.Phone == "" {
		return "Phone number is required", false
	}
	return "", true
}

func HandleRegister(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userData UserData
		if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid request format",
			})
			return
		}

		// Validate user data
		if msg, valid := validateUserData(userData); !valid {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": msg,
			})
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Check if email already exists
		var exists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM Users WHERE Email = ?)", userData.Email).Scan(&exists)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if exists {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Email already registered",
			})
			return
		}

		// Hash password
		hashedPassword, err := HashPassword(userData.Password)
		if err != nil {
			http.Error(w, "Error processing password", http.StatusInternalServerError)
			return
		}

		// Insert user
		result, err := tx.Exec(`
			INSERT INTO Users (Username, Email, PasswordHash, UserType, CreatedAt, IsActive)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, true)`,
			userData.Email, userData.Email, hashedPassword, userData.UserType)
		
		if err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting user ID", http.StatusInternalServerError)
			return
		}

		// Insert additional user info based on user type
		switch userData.UserType {
		case "parent":
			_, err = tx.Exec(`
				INSERT INTO Parents (UserID, FirstName, LastName, PhoneNumber, IsActive)
				VALUES (?, ?, ?, ?, true)`,
				userID, userData.FirstName, userData.LastName, userData.Phone)
		case "driver":
			_, err = tx.Exec(`
				INSERT INTO Drivers (UserID, FirstName, LastName, PhoneNumber, IsActive)
				VALUES (?, ?, ?, ?, true)`,
				userID, userData.FirstName, userData.LastName, userData.Phone)
		}

		if err != nil {
			http.Error(w, "Error creating user profile", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			http.Error(w, "Error completing registration", http.StatusInternalServerError)
			return
		}

		// Generate JWT token
		token, err := GenerateToken(int(userID), userData.UserType)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"token":    token,
			"userType": userData.UserType,
		})
	}
}

func HandlePasswordChange(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context (set by auth middleware)
		userID, ok := r.Context().Value("userID").(int)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Parse request
		var req struct {
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Validate new password
		if len(req.NewPassword) < 8 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Password must be at least 8 characters long",
			})
			return
		}

		// Verify current password
		var passwordHash string
		var resetRequired bool
		err := db.QueryRow("SELECT PasswordHash, PasswordResetRequired FROM Users WHERE UserID = ?", userID).
			Scan(&passwordHash, &resetRequired)
		if err != nil {
			log.Printf("Error fetching user password data: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Check current password
		if !CheckPasswordHash(req.CurrentPassword, passwordHash) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Current password is incorrect",
			})
			return
		}

		// Hash new password
		hashedNewPassword, err := HashPassword(req.NewPassword)
		if err != nil {
			log.Printf("Error hashing new password: %v", err)
			http.Error(w, "Error processing password", http.StatusInternalServerError)
			return
		}

		// Update password in database
		_, err = db.Exec("UPDATE Users SET PasswordHash = ?, PasswordResetRequired = false WHERE UserID = ?",
			hashedNewPassword, userID)
		if err != nil {
			log.Printf("Error updating password: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Password changed successfully",
		})
	}
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Authorization token required",
			})
			return
		}

		// Remove 'Bearer ' prefix if present
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid or expired token",
			})
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		ctx = context.WithValue(ctx, "userType", claims.UserType)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
