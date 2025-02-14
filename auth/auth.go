package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
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

func validateCredentials(creds Credentials) (string, bool) {
	if creds.Email == "" {
		return "Email is required", false
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
			UserID       int
			PasswordHash string
			UserType     string
		}

		err := db.QueryRow(`
			SELECT UserID, PasswordHash, UserType 
			FROM Users 
			WHERE Email = ? AND UserType = ? AND IsActive = true`,
			creds.Email, creds.UserType).Scan(&user.UserID, &user.PasswordHash, &user.UserType)

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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
