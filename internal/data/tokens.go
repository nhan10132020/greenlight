package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/nhan10132020/greenlight/internal/validator"
	"gorm.io/gorm"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token" gorm:"-"`
	Hash      []byte    `json:"-" gorm:"column:hash"`
	UserID    int64     `json:"-" gorm:"column:user_id"`
	Expiry    time.Time `json:"expiry" gorm:"column:expiry"`
	Scope     string    `json:"-" gorm:"column:scope"`
}

func (Token) TableName() string { return "tokens" }

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	// 128 bits (16 bytes) of entropy
	randomeBytes := make([]byte, 16)

	// fill the byte slice with random bytes from operating systems's CSPRNG(cryptographically secure random number generator)
	_, err := rand.Read(randomeBytes)
	if err != nil {
		return nil, err
	}

	// encode byte slice to a base-32-encoded string with entropy of 16 bytes.
	// the length of plaintext is 26 due to base-32 string encoded of 16 bytes
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomeBytes)

	// generate hash of plaintext by SHA-256 algorithm
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// Check that the plaintext token has been provided and is exactly
// 26 bytes long
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *gorm.DB
}

func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	err = m.Insert(token)
	return token, err
}

func (m TokenModel) Insert(token *Token) error {
	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.DB.WithContext(ctx).Create(token).Error; err != nil {
		return err
	}

	return nil
}

func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.DB.WithContext(ctx).Where("scope = ? AND user_id = ?", scope, userID).Delete(&Token{}).Error; err != nil {
		return err
	}

	return nil
}
