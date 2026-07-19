package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are embedded in access/refresh JWTs.
type Claims struct {
	OrgID uuid.UUID `json:"org_id"`
	Role  string    `json:"role"`
	Kind  string    `json:"kind"` // "access" | "refresh"
	jwt.RegisteredClaims
}

// TokenManager issues and validates JWTs.
type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret []byte, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *TokenManager) issue(userID, orgID uuid.UUID, role, kind string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		OrgID: orgID,
		Role:  role,
		Kind:  kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) Access(userID, orgID uuid.UUID, role string) (string, error) {
	return m.issue(userID, orgID, role, "access", m.accessTTL)
}

func (m *TokenManager) Refresh(userID, orgID uuid.UUID, role string) (string, error) {
	return m.issue(userID, orgID, role, "refresh", m.refreshTTL)
}

func (m *TokenManager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// SignRID produces an opaque, tamper-resistant tracking id for a campaign target.
// Format: base64url(campaignTargetID) . base64url(hmac). It is unguessable without
// the server RID secret, preventing target enumeration.
func SignRID(secret []byte, campaignTargetID uuid.UUID) string {
	idBytes, _ := campaignTargetID.MarshalBinary()
	mac := hmac.New(sha256.New, secret)
	mac.Write(idBytes)
	sig := mac.Sum(nil)[:12]
	return base64.RawURLEncoding.EncodeToString(idBytes) + "." +
		base64.RawURLEncoding.EncodeToString(sig)
}

// VerifyRID validates a tracking id and returns the campaign target id.
func VerifyRID(secret []byte, rid string) (uuid.UUID, error) {
	dot := -1
	for i := 0; i < len(rid); i++ {
		if rid[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return uuid.Nil, errors.New("malformed rid")
	}
	idBytes, err := base64.RawURLEncoding.DecodeString(rid[:dot])
	if err != nil {
		return uuid.Nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(rid[dot+1:])
	if err != nil {
		return uuid.Nil, err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(idBytes)
	want := mac.Sum(nil)[:12]
	if !hmac.Equal(sig, want) {
		return uuid.Nil, errors.New("rid signature mismatch")
	}
	id, err := uuid.FromBytes(idBytes)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
