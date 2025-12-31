package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/steipete/gogcli/internal/config"
	"golang.org/x/term"
)

type Store interface {
	Keys() ([]string, error)
	SetToken(email string, tok Token) error
	GetToken(email string) (Token, error)
	DeleteToken(email string) error
	ListTokens() ([]Token, error)
	GetDefaultAccount() (string, error)
	SetDefaultAccount(email string) error
}

type KeyringStore struct {
	ring keyring.Keyring
}

type Token struct {
	Email        string    `json:"email"`
	Services     []string  `json:"services,omitempty"`
	Scopes       []string  `json:"scopes,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	RefreshToken string    `json:"-"`
}

const keyringPasswordEnv = "GOG_KEYRING_PASSWORD" //nolint:gosec // env var name, not a credential

func fileKeyringPasswordFuncFrom(password string, isTTY bool) keyring.PromptFunc {
	if password != "" {
		return keyring.FixedStringPrompt(password)
	}

	if isTTY {
		return keyring.TerminalPrompt
	}

	return func(_ string) (string, error) {
		return "", fmt.Errorf("no TTY available for keyring file backend password prompt; set %s", keyringPasswordEnv)
	}
}

func fileKeyringPasswordFunc() keyring.PromptFunc {
	return fileKeyringPasswordFuncFrom(os.Getenv(keyringPasswordEnv), term.IsTerminal(int(os.Stdin.Fd())))
}

func OpenDefault() (Store, error) {
	// On Linux/WSL/containers, OS keychains (secret-service/kwallet) may be unavailable.
	// In that case github.com/99designs/keyring falls back to the "file" backend,
	// which *requires* both a directory and a password prompt function.
	keyringDir, err := config.EnsureKeyringDir()
	if err != nil {
		return nil, err
	}

	ring, err := keyring.Open(keyring.Config{
		ServiceName:              config.AppName,
		KeychainTrustApplication: runtime.GOOS == "darwin",
		FileDir:                  keyringDir,
		FilePasswordFunc:         fileKeyringPasswordFunc(),
	})
	if err != nil {
		return nil, err
	}
	return &KeyringStore{ring: ring}, nil
}

func (s *KeyringStore) Keys() ([]string, error) {
	return s.ring.Keys()
}

type storedToken struct {
	RefreshToken string    `json:"refresh_token"`
	Services     []string  `json:"services,omitempty"`
	Scopes       []string  `json:"scopes,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

func (s *KeyringStore) SetToken(email string, tok Token) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}
	if tok.RefreshToken == "" {
		return fmt.Errorf("missing refresh token")
	}
	if tok.CreatedAt.IsZero() {
		tok.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(storedToken{
		RefreshToken: tok.RefreshToken,
		Services:     tok.Services,
		Scopes:       tok.Scopes,
		CreatedAt:    tok.CreatedAt,
	})
	if err != nil {
		return err
	}

	return s.ring.Set(keyring.Item{
		Key:  tokenKey(email),
		Data: payload,
	})
}

func (s *KeyringStore) GetToken(email string) (Token, error) {
	email = normalize(email)
	if email == "" {
		return Token{}, fmt.Errorf("missing email")
	}
	it, err := s.ring.Get(tokenKey(email))
	if err != nil {
		return Token{}, err
	}
	var st storedToken
	if err := json.Unmarshal(it.Data, &st); err != nil {
		return Token{}, err
	}
	return Token{
		Email:        email,
		Services:     st.Services,
		Scopes:       st.Scopes,
		CreatedAt:    st.CreatedAt,
		RefreshToken: st.RefreshToken,
	}, nil
}

func (s *KeyringStore) DeleteToken(email string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}
	return s.ring.Remove(tokenKey(email))
}

func (s *KeyringStore) ListTokens() ([]Token, error) {
	keys, err := s.Keys()
	if err != nil {
		return nil, err
	}
	out := make([]Token, 0)
	for _, k := range keys {
		email, ok := ParseTokenKey(k)
		if !ok {
			continue
		}
		tok, err := s.GetToken(email)
		if err != nil {
			return nil, err
		}
		out = append(out, tok)
	}
	return out, nil
}

func ParseTokenKey(k string) (email string, ok bool) {
	const prefix = "token:"
	if !strings.HasPrefix(k, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(k, prefix)
	if strings.TrimSpace(rest) == "" {
		return "", false
	}
	return rest, true
}

func tokenKey(email string) string {
	return fmt.Sprintf("token:%s", email)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

const defaultAccountKey = "default_account"

func (s *KeyringStore) GetDefaultAccount() (string, error) {
	it, err := s.ring.Get(defaultAccountKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", nil
		}
		return "", err
	}
	return string(it.Data), nil
}

func (s *KeyringStore) SetDefaultAccount(email string) error {
	email = normalize(email)
	if email == "" {
		return fmt.Errorf("missing email")
	}
	return s.ring.Set(keyring.Item{
		Key:  defaultAccountKey,
		Data: []byte(email),
	})
}
