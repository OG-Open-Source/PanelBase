package token_store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	logger "github.com/OG-Open-Source/PanelBase/internal/logging"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	bolt "go.etcd.io/bbolt"
)

// Constants for database file and bucket names.
const (
	dbFileName = "configs/tokens.db"
)

// Variables for database instance and bucket names (as byte slices).
var (
	db                *bolt.DB // Database instance
	dbPath            string   // Full path to the database file
	dbMutex           sync.Mutex
	tokensBucketName  = []byte("tokens")         // Bucket for storing token details
	revokedBucketName = []byte("revoked_tokens") // Bucket for storing revoked token JTIs
)

// TokenInfo defines the structure for storing token metadata in the database.
// Moved assumption here for clarity, ASSUMING this matches models/token.go if it exists
type TokenInfo struct {
	UserID      string                 `json:"user_id"`
	Audience    string                 `json:"audience"` // e.g., "api", "session"
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Scopes      models.UserPermissions `json:"scopes,omitempty"`
	IssuedAt    models.RFC3339Time     `json:"issued_at"`
	ExpiresAt   models.RFC3339Time     `json:"expires_at"`
	LastUsed    models.RFC3339Time     `json:"last_used,omitempty"`
	Revoked     bool                   `json:"revoked,omitempty"` // Used internally, not directly stored in tokens bucket
	CreatedAt   models.RFC3339Time     `json:"created_at"`        // Added CreatedAt field
}

// InitStore initializes the token store database.
// It ensures the database file exists and the required buckets are created.
// func InitStore(configPath string) error { // configPath removed, using dbFileName constant
func InitStore() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db != nil {
		return nil // Already initialized
	}

	// Determine absolute path for the database file
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	dbPath = filepath.Join(wd, dbFileName)

	// Ensure the directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0750); err != nil {
		return fmt.Errorf("failed to create token store directory '%s': %w", dbDir, err)
	}

	// Open the database file.
	dbInstance, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Printf("FATAL: Failed to open token store database '%s': %v", dbPath, err)
		return fmt.Errorf("failed to open token store database '%s': %w", dbPath, err)
	}
	db = dbInstance

	logger.Printf("TOKEN_STORE", "INIT", "Database opened at: %s", dbPath)

	// Ensure buckets exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(tokensBucketName)
		if err != nil {
			return fmt.Errorf("failed to create token bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists(revokedBucketName)
		if err != nil {
			return fmt.Errorf("failed to create revoked bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("FATAL: Failed to ensure buckets in token store: %v", err)
		CloseStore() // Ensure DB is closed on error
		return err
	}
	logger.Printf("TOKEN_STORE", "INIT", "Required buckets ensured: %s, %s", string(tokensBucketName), string(revokedBucketName))

	return nil
}

// CloseStore closes the database connection.
func CloseStore() {
	if db != nil {
		db.Close()
		fmt.Println("Token store closed.")
	}
}

// --- Core Methods ---

// StoreToken stores the token metadata associated with a JTI.
func StoreToken(jti string, info TokenInfo) error {
	if db == nil {
		return fmt.Errorf("token store not initialized")
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(tokensBucketName)
		if b == nil {
			return fmt.Errorf("bucket %s not found", string(tokensBucketName))
		}

		// Marshal the TokenInfo struct to JSON
		jsonData, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal token info: %w", err)
		}

		// Store jti -> jsonData
		if err := b.Put([]byte(jti), jsonData); err != nil {
			return fmt.Errorf("failed to store token info for jti %s: %w", jti, err)
		}
		return nil
	})
}

// GetTokenInfo retrieves the metadata for a specific token JTI.
// It returns the TokenInfo, a boolean indicating if found, and an error.
func GetTokenInfo(jti string) (info TokenInfo, found bool, err error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db == nil {
		return TokenInfo{}, false, fmt.Errorf("token store not initialized")
	}

	err = db.View(func(tx *bolt.Tx) error { // Use the named return variables
		tokensBucket := tx.Bucket(tokensBucketName)
		if tokensBucket == nil {
			return fmt.Errorf("bucket %s not found", string(tokensBucketName))
		}
		revokedBucket := tx.Bucket(revokedBucketName)
		if revokedBucket == nil {
			logger.ErrorPrintf("TOKEN_STORE", "GET", "Revoked token bucket not found during GetTokenInfo!")
			return fmt.Errorf("internal error: revoked bucket missing")
		}

		jsonData := tokensBucket.Get([]byte(jti))
		if jsonData == nil {
			found = false
			return nil // Not found is not an error for View transaction
		}

		if errUnmarshal := json.Unmarshal(jsonData, &info); errUnmarshal != nil {
			logger.ErrorPrintf("TOKEN_STORE", "GET", "Failed to unmarshal token info for %s: %v", jti, errUnmarshal)
			// Return the unmarshal error specifically
			return fmt.Errorf("failed to unmarshal token info for %s: %w", jti, errUnmarshal)
		}
		found = true
		logger.DebugPrintf("TOKEN_STORE", "GET", "JTI: %s, Retrieved Info: %+v", jti, info)

		// Check if token is revoked using the already fetched revokedBucket
		revokedData := revokedBucket.Get([]byte(jti))
		if revokedData != nil {
			info.Revoked = true // Set the Revoked field on the retrieved info
		}

		logger.DebugPrintf("TOKEN_STORE", "GET_RETURN", "JTI: %s, Returning Info: %+v", jti, info)
		return nil // View transaction successful
	})

	// Error from db.View() is returned automatically via named return `err`
	// Info and found are also set correctly within the transaction
	return info, found, err
}

// RevokeToken marks a JTI as revoked by adding it to the revoked bucket with the current timestamp.
func RevokeToken(jti string) error {
	if db == nil {
		return fmt.Errorf("token store not initialized")
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(revokedBucketName)
		if b == nil {
			return fmt.Errorf("bucket %s not found", string(revokedBucketName))
		}

		// Store the revocation time (or just a marker)
		// Storing timestamp allows potential cleanup based on time later
		revocationTime := time.Now().UTC().Format(time.RFC3339)
		return b.Put([]byte(jti), []byte(revocationTime))
	})
}

// IsTokenRevoked checks if a JTI exists in the revoked bucket.
func IsTokenRevoked(jti string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("token store not initialized")
	}
	revoked := false
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(revokedBucketName)
		if b == nil {
			// If bucket doesn't exist, token cannot be revoked (treat as error or false?)
			// Let's assume bucket should exist from InitStore
			return fmt.Errorf("bucket %s not found", string(revokedBucketName))
		}

		val := b.Get([]byte(jti))
		if val != nil {
			// Key exists, meaning the token is revoked
			revoked = true
		}
		return nil
	})

	if err != nil {
		return false, err
	}
	return revoked, nil
}

// GetUserTokens retrieves all non-revoked API tokens for a given user ID.
func GetUserTokens(userID string) (tokens []TokenInfo, count int, err error) {
	if db == nil {
		return nil, 0, fmt.Errorf("token store not initialized")
	}
	err = db.View(func(tx *bolt.Tx) error {
		tokensBucket := tx.Bucket(tokensBucketName)
		revokedBucket := tx.Bucket(revokedBucketName)
		if tokensBucket == nil || revokedBucket == nil {
			return fmt.Errorf("required buckets not found")
		}

		return tokensBucket.ForEach(func(k, v []byte) error {
			tokenID := string(k)
			if revokedBucket.Get(k) != nil {
				return nil // Skip revoked
			}

			var info TokenInfo
			if err := json.Unmarshal(v, &info); err != nil {
				logger.ErrorPrintf("TOKEN_STORE", "LIST", "Error unmarshaling token info for JTI %s: %v. Skipping.", tokenID, err)
				return nil
			}

			// Check if it belongs to the user AND is an API token
			if info.UserID == userID && info.Audience == "api" {
				tokens = append(tokens, info)
				count++
			}

			return nil
		})
	})

	if err != nil {
		return nil, 0, fmt.Errorf("error retrieving user tokens: %w", err)
	}
	if tokens == nil {
		tokens = []TokenInfo{}
	}
	return tokens, count, nil
}

// TODO: Implement CleanupExpiredTokens (important!)
// TODO: Implement Replay Protection methods if needed (MarkJTIUsed, IsJTIUsed)
