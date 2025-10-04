package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"onlineClinic/config"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

type Claims struct {
	UserID      int    `json:"user_id"`
	PhoneNumber string `json:"phone_number"`
	IsDoctor    bool   `json:"is_doctor"`
	IsPatient   bool   `json:"is_patient"`
	jwt.StandardClaims
}

type contextKey string

const UserClaimsKey contextKey = "userClaims"

func GenerateToken(userID int, phoneNumber string, isDoctor bool, isPatient bool) (string, error) {
	// log.Printf("Generating token - UserID: %d, Phone: %s, IsDoctor: %v, IsPatient: %v", userID, phoneNumber, isDoctor, isPatient)

	if isDoctor && isPatient {
		// log.Printf("Warning: User cannot be both doctor and patient - UserID: %d", userID)
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	// log.Printf("Token expiration set to: %v", expirationTime)

	claims := Claims{
		UserID:      userID,
		PhoneNumber: phoneNumber,
		IsDoctor:    isDoctor,
		IsPatient:   isPatient,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// log.Printf("Token claims created - ExpiresAt: %v, IssuedAt: %v", time.Unix(claims.ExpiresAt, 0), time.Unix(claims.IssuedAt, 0))

	signedToken, err := token.SignedString([]byte(config.Cfg.JWTSecret))
	if err != nil {
		// log.Printf("Failed to sign token for UserID %d: %v", userID, err)
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	// tokenPreview := signedToken
	if len(signedToken) > 10 {
		// tokenPreview = signedToken[:5] + "..." + signedToken[len(signedToken)-5:]
	}
	// log.Printf("Successfully generated token for UserID %d: %s", userID, tokenPreview)

	return signedToken, nil
}

func VerifyToken(tokenString string) (*Claims, error) {
	// log.Print("Starting token verification")

	// tokenPreview := tokenString
	if len(tokenString) > 10 {
		// tokenPreview = tokenString[:5] + "..." + tokenString[len(tokenString)-5:]
	}
	// log.Printf("Verifying token: %s", tokenPreview)

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// log.Printf("Invalid signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.Cfg.JWTSecret), nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			// log.Printf("Token validation error: %v (error flags: %d)", err, ve.Errors)
			switch {
			case ve.Errors&jwt.ValidationErrorMalformed != 0:
				return nil, fmt.Errorf("malformed token")
			case ve.Errors&jwt.ValidationErrorExpired != 0:
				return nil, fmt.Errorf("token expired")
			case ve.Errors&jwt.ValidationErrorNotValidYet != 0:
				return nil, fmt.Errorf("token not yet valid")
			default:
				return nil, fmt.Errorf("token validation error: %v", err)
			}
		}
		// log.Printf("Failed to parse token: %v", err)
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		// log.Print("Token is invalid")
		return nil, errors.New("invalid token")
	}

	// log.Printf("Token verified successfully - UserID: %d, IsDoctor: %v, IsPatient: %v", claims.UserID, claims.IsDoctor, claims.IsPatient)

	if time.Unix(claims.ExpiresAt, 0).Before(time.Now()) {
		// log.Printf("Token expired at %v", time.Unix(claims.ExpiresAt, 0))
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

func setCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin, Accept")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("Processing auth middleware for request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all responses
		setCORSHeaders(w, "https://kashan-clininc.liara.run")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// log.Printf("No Authorization header present in request from IP: %s", r.RemoteAddr)
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			// log.Printf("Invalid token format in request from IP: %s", r.RemoteAddr)
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		claims, err := VerifyToken(bearerToken[1])
		if err != nil {
			// log.Printf("Token verification failed for request from IP %s: %v", r.RemoteAddr, err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// log.Printf("Request authenticated - UserID: %d, IsDoctor: %v, IsPatient: %v, Path: %s", claims.UserID, claims.IsDoctor, claims.IsPatient, r.URL.Path)

		ctx := SetUserClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func IsPatient(ctx context.Context) bool {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return false
	}
	return claims.IsPatient
}

func IsDoctorOrPatient(ctx context.Context) bool {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return false
	}
	return claims.IsDoctor || claims.IsPatient
}

func SetUserClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, UserClaimsKey, claims)
}

func GetUserClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*Claims)
	return claims, ok
}

func GetUserID(ctx context.Context) (int, bool) {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return 0, false
	}
	return claims.UserID, true
}

func IsDoctor(ctx context.Context) bool {
	claims, ok := GetUserClaims(ctx)
	if !ok {
		return false
	}
	return claims.IsDoctor
}

func DoctorOrPatientAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("Checking doctor/patient authorization for request: %s %s", r.Method, r.URL.Path)

		// Set CORS headers for all responses
		setCORSHeaders(w, "https://kashan-clininc.liara.run")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			// log.Print("No authorization token provided")
			http.Error(w, "No authorization token provided", http.StatusUnauthorized)
			return
		}

		// Remove 'Bearer ' prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		// Verify token and get claims
		claims, err := VerifyToken(token)
		if err != nil {
			// log.Printf("Token verification failed: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Check if user is either a doctor or a patient
		if !claims.IsDoctor && !claims.IsPatient {
			// log.Printf("Unauthorized access attempt - User ID: %d, IsDoctor: %v, IsPatient: %v", claims.UserID, claims.IsDoctor, claims.IsPatient)
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}

		// log.Printf("Authorization successful - User ID: %d, IsDoctor: %v, IsPatient: %v", claims.UserID, claims.IsDoctor, claims.IsPatient)

		// Add claims to request context
		ctx := SetUserClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func PatientAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all responses
		setCORSHeaders(w, "https://kashan-clininc.liara.run")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "No authorization token provided", http.StatusUnauthorized)
			return
		}

		// Remove 'Bearer ' prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		// Verify token and get claims
		claims, err := VerifyToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Check if token is for a patient
		if claims.IsDoctor {
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}

		// Add claims to request context
		ctx := SetUserClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func DoctorAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all responses
		setCORSHeaders(w, "https://kashan-clininc.liara.run")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "No authorization token provided", http.StatusUnauthorized)
			return
		}

		// Remove 'Bearer ' prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		// Verify token and get claims
		claims, err := VerifyToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Check if token is for a doctor
		if !claims.IsDoctor {
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}

		// Check resource ownership for paths containing doctor ID
		vars := mux.Vars(r)
		if doctorID, exists := vars["id"]; exists {
			requestedID, err := strconv.Atoi(doctorID)
			if err != nil {
				http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
				return
			}

			// Verify the authenticated doctor is modifying their own availability
			if claims.UserID != requestedID {
				// log.Printf("Unauthorized attempt: Doctor %d trying to modify Doctor %d's availability", claims.UserID, requestedID)
				http.Error(w, "Unauthorized: Can only modify own availability", http.StatusForbidden)
				return
			}
		}

		// Add claims to request context
		ctx := SetUserClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
