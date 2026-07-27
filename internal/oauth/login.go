// Package oauth implements AtomGit OAuth2 authorization-code login (browser + local callback).
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
)

// Defaults match AtomCode / AtomGit OAuth app (see project oauth reference); override with env.
const (
	defaultClientID     = "b9956e5327e544578128af8979ba3ccb"
	defaultClientSecret = "756ef00061884c7aa1ac64bd4eae3be7"
	defaultRedirectPort = "8765"

	authorizeURL = "https://atomgit.com/oauth/authorize"
	tokenURL     = "https://atomgit.com/oauth/token"
	userURL      = "https://atomgit.com/api/v5/user"
	scopes       = "user_info projects"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type userResponse struct {
	ID        interface{} `json:"id"`
	Login     string      `json:"login"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	AvatarURL string      `json:"avatar_url"`
}

func clientID() string {
	if s := strings.TrimSpace(os.Getenv("AG_OAUTH_CLIENT_ID")); s != "" {
		return s
	}
	return defaultClientID
}

func clientSecret() string {
	if s := strings.TrimSpace(os.Getenv("AG_OAUTH_CLIENT_SECRET")); s != "" {
		return s
	}
	return defaultClientSecret
}

func redirectPort() string {
	if s := strings.TrimSpace(os.Getenv("AG_OAUTH_REDIRECT_PORT")); s != "" {
		return s
	}
	return defaultRedirectPort
}

func redirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%s/callback", redirectPort())
}

// LoginResult contains tokens and user identity for persisting after authorization.
type LoginResult struct {
	AccessToken  string
	Login        string
	Name         string
	Email        string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
}

// Login runs the browser OAuth flow and returns tokens and AtomGit login name.
func Login(ctx context.Context) (*LoginResult, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	redir := redirectURI()
	authURL := buildAuthorizeURL(clientID(), redir, state)

	resultCh := make(chan callbackResult, 1)
	var once sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		if errStr := q.Get("error"); errStr != "" {
			desc := q.Get("error_description")
			once.Do(func() {
				resultCh <- callbackResult{err: fmt.Errorf("oauth error: %s %s", errStr, desc)}
			})
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}
		code := q.Get("code")
		st := q.Get("state")
		if code == "" {
			once.Do(func() {
				resultCh <- callbackResult{err: fmt.Errorf("no authorization code in callback")}
			})
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, successHTML)
		once.Do(func() {
			resultCh <- callbackResult{code: code, state: st}
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	addr := net.JoinHostPort("127.0.0.1", redirectPort())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w (free the port or set AG_OAUTH_REDIRECT_PORT)", addr, err)
	}

	go func() { _ = srv.Serve(ln) }()
	defer func() {
		shctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shctx)
	}()

	fmt.Println("Opening browser for AtomGit authorization…")
	fmt.Println("If it does not open, visit:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Printf("Waiting for callback on %s …\n", redir)

	if err := browser.NewAsyncOpener()(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		if res.state != state {
			return nil, fmt.Errorf("oauth state mismatch (possible CSRF)")
		}
		tok, err := exchangeCode(ctx, clientID(), clientSecret(), redir, res.code)
		if err != nil {
			return nil, err
		}
		user, err := fetchUser(ctx, tok.AccessToken)
		if err != nil {
			return nil, err
		}
		if user.Login == "" {
			return nil, fmt.Errorf("user API returned empty login")
		}
		tt := tok.TokenType
		if tt == "" {
			tt = "Bearer"
		}
		return &LoginResult{
			AccessToken:  tok.AccessToken,
			Login:        user.Login,
			Name:         user.Name,
			Email:        user.Email,
			RefreshToken: tok.RefreshToken,
			ExpiresIn:    tok.ExpiresIn,
			TokenType:    tt,
		}, nil

	case <-ctx.Done():
		return nil, fmt.Errorf("login cancelled or timed out: %w", ctx.Err())
	}
}

type callbackResult struct {
	code  string
	state string
	err   error
}

func randomState() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func buildAuthorizeURL(clientID, redirectURI, state string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("state", state)
	v.Set("scope", scopes)
	return authorizeURL + "?" + v.Encode()
}

func exchangeCode(ctx context.Context, id, secret, redir, code string) (*tokenResponse, error) {
	body := url.Values{}
	body.Set("client_id", id)
	body.Set("client_secret", secret)
	body.Set("code", code)
	body.Set("redirect_uri", redir)
	body.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token endpoint %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var tr tokenResponse
	if err := json.Unmarshal(b, &tr); err != nil {
		return nil, fmt.Errorf("parse token JSON: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	return &tr, nil
}

// RefreshedTokens is returned by RefreshAccessToken.
type RefreshedTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
}

// RefreshAccessToken uses a refresh_token grant to obtain new tokens.
func RefreshAccessToken(ctx context.Context, refreshToken string) (*RefreshedTokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh_token is empty")
	}
	body := url.Values{}
	body.Set("client_id", clientID())
	body.Set("client_secret", clientSecret())
	body.Set("refresh_token", refreshToken)
	body.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token endpoint %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var tr tokenResponse
	if err := json.Unmarshal(b, &tr); err != nil {
		return nil, fmt.Errorf("parse token JSON: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	tt := tr.TokenType
	if tt == "" {
		tt = "Bearer"
	}
	return &RefreshedTokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
		TokenType:    tt,
	}, nil
}

func fetchUser(ctx context.Context, accessToken string) (*userResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("user endpoint %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var u userResponse
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, fmt.Errorf("parse user JSON: %w", err)
	}
	return &u, nil
}

const successHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>AtomGit Login</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#1a1a2e;color:#eee}
.container{text-align:center;padding:2rem}h1{color:#7c3aed;margin:0}p{color:#888}.ok{color:#22c55e;font-size:4rem}</style></head>
<body><div class="container"><div class="ok">✓</div><h1>Authorization successful</h1><p>You can close this window and return to the terminal.</p></div></body></html>`
