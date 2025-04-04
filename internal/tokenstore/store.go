package tokenstore

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models" // For UserPermissions
	bolt "go.etcd.io/bbolt"
)

// Constants for database file and bucket names
const (
	dbFileName        = "configs/tokens.db"
	tokensBucketName  = "tokens"
	revokedBucketName = "revoked_tokens"
)

// Bucket names
var (
	tokenInfoBucket    = []byte("TokenInfo")
	revokedTokenBucket = []byte("RevokedTokens")
	// TODO: Add user index bucket if needed: userTokensBucket = []byte("UserTokensIndex")
)

var db *bolt.DB // Global database connection variable

// TokenInfo holds the metadata associated with a JTI.
// This is stored as the value in the TokenInfoBucket.
type TokenInfo struct {
	UserID    string                 `json:"user_id"`
	Name      string                 `json:"name"`
	Audience  string                 `json:"audience"` // "web" or "api"
	Scopes    models.UserPermissions `json:"scopes"`
	CreatedAt models.RFC3339Time     `json:"created_at"`
	ExpiresAt models.RFC3339Time     `json:"expires_at"`
}

// InitStore initializes the BoltDB database and creates necessary buckets.
func InitStore() error {
	var err error
	// Ensure the directory exists
	dbDirResolved := filepath.Dir(dbFileName) // Use filepath.Dir
	if err := os.MkdirAll(dbDirResolved, 0750); err != nil {
		return fmt.Errorf("failed to create token store directory '%s': %w", dbDirResolved, err)
	}

	dbPath := dbFileName // Use the constant directly
	db, err = bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("failed to open token store database '%s': %w", dbPath, err)
	}
	log.Printf("Token store database opened at: %s", dbPath)

	// Ensure buckets exist
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(tokensBucketName))
		if err != nil {
			return fmt.Errorf("failed to create bucket '%s': %w", tokensBucketName, err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(revokedBucketName))
		if err != nil {
			return fmt.Errorf("failed to create bucket '%s': %w", revokedBucketName, err)
		}
		log.Printf("Required token store buckets ensured: %s, %s", tokensBucketName, revokedBucketName)
		return nil
	})
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
		b := tx.Bucket([]byte(tokensBucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", tokensBucketName)
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

// GetTokenInfo retrieves token metadata by JTI.
func GetTokenInfo(jti string) (TokenInfo, bool, error) {
	if db == nil {
		return TokenInfo{}, false, fmt.Errorf("token store not initialized")
	}
	var info TokenInfo
	found := false
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tokensBucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", tokensBucketName)
		}

		jsonData := b.Get([]byte(jti))
		if jsonData == nil {
			return nil // Not found
		}

		if err := json.Unmarshal(jsonData, &info); err != nil {
			return fmt.Errorf("failed to unmarshal token info for jti %s: %w", jti, err)
		}
		found = true
		// Log the retrieved info for debugging
		log.Printf("[DEBUG TOKENSTORE GET] JTI: %s, Retrieved Info: %+v", jti, info)
		return nil
	})

	if err != nil {
		return TokenInfo{}, false, err
	}
	// Another log before returning, just in case
	if found {
		log.Printf("[DEBUG TOKENSTORE GET RET] JTI: %s, Returning Info: %+v", jti, info)
	}
	return info, found, nil
}

// RevokeToken marks a JTI as revoked by adding it to the revoked bucket with the current timestamp.
func RevokeToken(jti string) error {
	if db == nil {
		return fmt.Errorf("token store not initialized")
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(revokedBucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", revokedBucketName)
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
		b := tx.Bucket([]byte(revokedBucketName))
		if b == nil {
			// If bucket doesn't exist, token cannot be revoked (treat as error or false?)
			// Let's assume bucket should exist from InitStore
			return fmt.Errorf("bucket %s not found", revokedBucketName)
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

// GetUserTokens retrieves all non-revoked API tokens for a specific user.
// It filters by UserID and ensures the Audience is "api".
// Returns slices of TokenInfo, corresponding JTIs, and an error.
func GetUserTokens(userID string) ([]TokenInfo, []string, error) {
	if db == nil {
		return nil, nil, fmt.Errorf("token store not initialized")
	}
	var tokensInfo []TokenInfo
	var tokenIDs []string
	err := db.View(func(tx *bolt.Tx) error {
		tokensBucket := tx.Bucket([]byte(tokensBucketName))
		revokedBucket := tx.Bucket([]byte(revokedBucketName))
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
				log.Printf("Error unmarshaling token info for JTI %s: %v. Skipping.", tokenID, err)
				return nil
			}

			// Check if it belongs to the user AND is an API token
			if info.UserID == userID && info.Audience == "api" {
				tokensInfo = append(tokensInfo, info)
				tokenIDs = append(tokenIDs, tokenID)
			}

			return nil
		})
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving user tokens: %w", err)
	}
	if tokensInfo == nil {
		tokensInfo = []TokenInfo{}
	}
	if tokenIDs == nil {
		tokenIDs = []string{}
	}
	return tokensInfo, tokenIDs, nil
}

// TODO: Implement GetUserTokens (requires index)
// TODO: Implement CleanupExpiredTokens (important!)
// TODO: Implement Replay Protection methods if needed (MarkJTIUsed, IsJTIUsed)
