package googleauth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"

	"github.com/steipete/gogcli/internal/authclient"
	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/secrets"
)

// ManageServerOptions configures the accounts management server.
type ManageServerOptions struct {
	Timeout               time.Duration
	Services              []Service
	ForceConsent          bool
	Client                string
	ListenAddr            string
	RedirectURI           string
	UpdateEmailReferences EmailReferenceUpdater
	ReadCredentials       func(client string) (config.ClientCredentials, error)
	EnsureKeychainAccess  func() error
}

var openDefaultStore func() (secrets.Store, error)

// StartManageServer starts the accounts management server and opens a browser.
func StartManageServer(ctx context.Context, opts ManageServerOptions) error {
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}

	listenAddr, err := normalizeListenAddr(opts.ListenAddr)
	if err != nil {
		return err
	}

	if validationErr := validateManagementListenAddr(listenAddr); validationErr != nil {
		return validationErr
	}

	if strings.TrimSpace(opts.RedirectURI) != "" {
		resolvedRedirectURI, normalizeErr := normalizeRedirectURI(opts.RedirectURI)
		if normalizeErr != nil {
			return normalizeErr
		}
		opts.RedirectURI = resolvedRedirectURI
	}

	if opts.UpdateEmailReferences == nil {
		return errEmailReferenceUpdaterRequired
	}

	store, err := openManageSecretsStore(ctx)
	if err != nil {
		return fmt.Errorf("failed to open secrets store: %w", err)
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer ln.Close()

	redirectURI := resolveServerRedirectURI(ln, opts.RedirectURI)

	ensureKeychainAccess := func(context.Context) error { return nil }
	if opts.EnsureKeychainAccess != nil {
		ensureKeychainAccess = func(context.Context) error {
			return opts.EnsureKeychainAccess()
		}
	}

	app, err := NewManagerApplication(ManagerOptions{
		Services:     opts.Services,
		ForceConsent: opts.ForceConsent,
		Client:       opts.Client,
		RedirectURI:  redirectURI,
	}, ManagerDependencies{
		Tokens:                store,
		ReadCredentials:       manageCredentialsReader(ctx, opts.ReadCredentials),
		UpdateEmailReferences: opts.UpdateEmailReferences,
		FetchIdentity:         fetchUserIdentityDefault,
		EnsureKeychainAccess:  ensureKeychainAccess,
		Random:                rand.Reader,
		OAuthEndpoint:         google.Endpoint,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:      app.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	resultCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	go func() {
		if serveErr := server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			select {
			case resultCh <- serveErr:
			default:
			}
		}
	}()

	url := listenerBaseURL(ln)

	fmt.Fprintln(os.Stderr, "Opening accounts manager in browser...")
	fmt.Fprintln(os.Stderr, "If the browser doesn't open, visit:", url)

	if strings.TrimSpace(opts.ListenAddr) != "" {
		fmt.Fprintf(os.Stderr, "Server listening on %s\n", ln.Addr().String())
	}
	_ = openBrowserFn(url)

	select {
	case serveErr := <-resultCh:
		return serveErr
	case <-ctx.Done():
		_ = server.Close()
		return nil
	}
}

func openManageSecretsStore(ctx context.Context) (secrets.Store, error) {
	if openDefaultStore != nil {
		return openDefaultStore()
	}

	store, err := authclient.OpenSecretsStore(ctx)
	if err != nil {
		return nil, fmt.Errorf("open accounts manager secrets store: %w", err)
	}

	return store, nil
}
