package docker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/image/docker/reference"
	"github.com/containers/image/pkg/docker/config"
	"github.com/containers/image/pkg/tlsclientconfig"
	"github.com/containers/image/types"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	dockerHostname = "docker.io"
	dockerRegistry = "registry-1.docker.io"

	systemPerHostCertDirPath = "/etc/docker/certs.d"

	resolvedPingV2URL       = "%s://%s/v2/"
	resolvedPingV1URL       = "%s://%s/v1/_ping"
	tagsPath                = "/v2/%s/tags/list"
	manifestPath            = "/v2/%s/manifests/%s"
	blobsPath               = "/v2/%s/blobs/%s"
	blobUploadPath          = "/v2/%s/blobs/uploads/"
	extensionsSignaturePath = "/extensions/v2/%s/signatures/%s"

	minimumTokenLifetimeSeconds = 60

	extensionSignatureSchemaVersion = 2        // extensionSignature.Version
	extensionSignatureTypeAtomic    = "atomic" // extensionSignature.Type
)

var (
	// ErrV1NotSupported is returned when we're trying to talk to a
	// docker V1 registry.
	ErrV1NotSupported = errors.New("can't talk to a V1 docker registry")
	// ErrUnauthorizedForCredentials is returned when the status code returned is 401
	ErrUnauthorizedForCredentials = errors.New("unable to retrieve auth token: invalid username/password")
)

// extensionSignature and extensionSignatureList come from github.com/openshift/origin/pkg/dockerregistry/server/signaturedispatcher.go:
// signature represents a Docker image signature.
type extensionSignature struct {
	Version int    `json:"schemaVersion"` // Version specifies the schema version
	Name    string `json:"name"`          // Name must be in "sha256:<digest>@signatureName" format
	Type    string `json:"type"`          // Type is optional, of not set it will be defaulted to "AtomicImageV1"
	Content []byte `json:"content"`       // Content contains the signature
}

// signatureList represents list of Docker image signatures.
type extensionSignatureList struct {
	Signatures []extensionSignature `json:"signatures"`
}

type bearerToken struct {
	Token     string    `json:"token"`
	ExpiresIn int       `json:"expires_in"`
	IssuedAt  time.Time `json:"issued_at"`
}

// dockerClient is configuration for dealing with a single Docker registry.
type dockerClient struct {
	// The following members are set by newDockerClient and do not change afterwards.
	ctx           *types.SystemContext
	registry      string
	username      string
	password      string
	client        *http.Client
	signatureBase signatureStorageBase
	scope         authScope
	// The following members are detected registry properties:
	// They are set after a successful detectProperties(), and never change afterwards.
	scheme             string // Empty value also used to indicate detectProperties() has not yet succeeded.
	challenges         []challenge
	supportsSignatures bool
	// The following members are private state for setupRequestAuth, both are valid if token != nil.
	token           *bearerToken
	tokenExpiration time.Time
}

type authScope struct {
	remoteName string
	actions    string
}

// this is cloned from docker/go-connections because upstream docker has changed
// it and make deps here fails otherwise.
// We'll drop this once we upgrade to docker 1.13.x deps.
func serverDefault() *tls.Config {
	return &tls.Config{
		// Avoid fallback to SSL protocols < TLS1.0
		MinVersion:               tls.VersionTLS10,
		PreferServerCipherSuites: true,
		CipherSuites:             tlsconfig.DefaultServerAcceptedCiphers,
	}
}

// dockerCertDir returns a path to a directory to be consumed by tlsclientconfig.SetupCertificates() depending on ctx and hostPort.
func dockerCertDir(ctx *types.SystemContext, hostPort string) string {
	if ctx != nil && ctx.DockerCertPath != "" {
		return ctx.DockerCertPath
	}
	var hostCertDir string
	if ctx != nil && ctx.DockerPerHostCertDirPath != "" {
		hostCertDir = ctx.DockerPerHostCertDirPath
	} else if ctx != nil && ctx.RootForImplicitAbsolutePaths != "" {
		hostCertDir = filepath.Join(ctx.RootForImplicitAbsolutePaths, systemPerHostCertDirPath)
	} else {
		hostCertDir = systemPerHostCertDirPath
	}
	return filepath.Join(hostCertDir, hostPort)
}

// newDockerClientFromRef returns a new dockerClient instance for refHostname (a host a specified in the Docker image reference, not canonicalized to dockerRegistry)
// “write” specifies whether the client will be used for "write" access (in particular passed to lookaside.go:toplevelFromSection)
func newDockerClientFromRef(ctx *types.SystemContext, ref dockerReference, write bool, actions string) (*dockerClient, error) {
	registry := reference.Domain(ref.ref)
	username, password, err := config.GetAuthentication(ctx, reference.Domain(ref.ref))
	if err != nil {
		return nil, errors.Wrapf(err, "error getting username and password")
	}
	sigBase, err := configuredSignatureStorageBase(ctx, ref, write)
	if err != nil {
		return nil, err
	}
	remoteName := reference.Path(ref.ref)

	return newDockerClientWithDetails(ctx, registry, username, password, actions, sigBase, remoteName)
}

// newDockerClientWithDetails returns a new dockerClient instance for the given parameters
func newDockerClientWithDetails(ctx *types.SystemContext, registry, username, password, actions string, sigBase signatureStorageBase, remoteName string) (*dockerClient, error) {
	hostName := registry
	if registry == dockerHostname {
		registry = dockerRegistry
	}
	tr := tlsclientconfig.NewTransport()
	tr.TLSClientConfig = serverDefault()

	// It is undefined whether the host[:port] string for dockerHostname should be dockerHostname or dockerRegistry,
	// because docker/docker does not read the certs.d subdirectory at all in that case.  We use the user-visible
	// dockerHostname here, because it is more symmetrical to read the configuration in that case as well, and because
	// generally the UI hides the existence of the different dockerRegistry.  But note that this behavior is
	// undocumented and may change if docker/docker changes.
	certDir := dockerCertDir(ctx, hostName)
	if err := tlsclientconfig.SetupCertificates(certDir, tr.TLSClientConfig); err != nil {
		return nil, err
	}

	if ctx != nil && ctx.DockerInsecureSkipTLSVerify {
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	return &dockerClient{
		ctx:           ctx,
		registry:      registry,
		username:      username,
		password:      password,
		client:        &http.Client{Transport: tr},
		signatureBase: sigBase,
		scope: authScope{
			actions:    actions,
			remoteName: remoteName,
		},
	}, nil
}

// CheckAuth validates the credentials by attempting to log into the registry
// returns an error if an error occcured while making the http request or the status code received was 401
func CheckAuth(ctx context.Context, sCtx *types.SystemContext, username, password, registry string) error {
	newLoginClient, err := newDockerClientWithDetails(sCtx, registry, username, password, "", nil, "")
	if err != nil {
		return errors.Wrapf(err, "error creating new docker client")
	}

	resp, err := newLoginClient.makeRequest(ctx, "GET", "/v2/", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorizedForCredentials
	default:
		return errors.Errorf("error occured with status code %q", resp.StatusCode)
	}
}

// makeRequest creates and executes a http.Request with the specified parameters, adding authentication and TLS options for the Docker client.
// The host name and schema is taken from the client or autodetected, and the path is relative to it, i.e. the path usually starts with /v2/.
func (c *dockerClient) makeRequest(ctx context.Context, method, path string, headers map[string][]string, stream io.Reader) (*http.Response, error) {
	if err := c.detectProperties(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s://%s%s", c.scheme, c.registry, path)
	return c.makeRequestToResolvedURL(ctx, method, url, headers, stream, -1, true)
}

// makeRequestToResolvedURL creates and executes a http.Request with the specified parameters, adding authentication and TLS options for the Docker client.
// streamLen, if not -1, specifies the length of the data expected on stream.
// makeRequest should generally be preferred.
// TODO(runcom): too many arguments here, use a struct
func (c *dockerClient) makeRequestToResolvedURL(ctx context.Context, method, url string, headers map[string][]string, stream io.Reader, streamLen int64, sendAuth bool) (*http.Response, error) {
	req, err := http.NewRequest(method, url, stream)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if streamLen != -1 { // Do not blindly overwrite if streamLen == -1, http.NewRequest above can figure out the length of bytes.Reader and similar objects without us having to compute it.
		req.ContentLength = streamLen
	}
	req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")
	for n, h := range headers {
		for _, hh := range h {
			req.Header.Add(n, hh)
		}
	}
	if c.ctx != nil && c.ctx.DockerRegistryUserAgent != "" {
		req.Header.Add("User-Agent", c.ctx.DockerRegistryUserAgent)
	}
	if sendAuth {
		if err := c.setupRequestAuth(req); err != nil {
			return nil, err
		}
	}
	logrus.Debugf("%s %s", method, url)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// we're using the challenges from the /v2/ ping response and not the one from the destination
// URL in this request because:
//
// 1) docker does that as well
// 2) gcr.io is sending 401 without a WWW-Authenticate header in the real request
//
// debugging: https://github.com/containers/image/pull/211#issuecomment-273426236 and follows up
func (c *dockerClient) setupRequestAuth(req *http.Request) error {
	if len(c.challenges) == 0 {
		return nil
	}
	schemeNames := make([]string, 0, len(c.challenges))
	for _, challenge := range c.challenges {
		schemeNames = append(schemeNames, challenge.Scheme)
		switch challenge.Scheme {
		case "basic":
			req.SetBasicAuth(c.username, c.password)
			return nil
		case "bearer":
			if c.token == nil || time.Now().After(c.tokenExpiration) {
				realm, ok := challenge.Parameters["realm"]
				if !ok {
					return errors.Errorf("missing realm in bearer auth challenge")
				}
				service, _ := challenge.Parameters["service"] // Will be "" if not present
				var scope string
				if c.scope.remoteName != "" && c.scope.actions != "" {
					scope = fmt.Sprintf("repository:%s:%s", c.scope.remoteName, c.scope.actions)
				}
				token, err := c.getBearerToken(req.Context(), realm, service, scope)
				if err != nil {
					return err
				}
				c.token = token
				c.tokenExpiration = token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token.Token))
			return nil
		default:
			logrus.Debugf("no handler for %s authentication", challenge.Scheme)
		}
	}
	logrus.Infof("None of the challenges sent by server (%s) are supported, trying an unauthenticated request anyway", strings.Join(schemeNames, ", "))
	return nil
}

func (c *dockerClient) getBearerToken(ctx context.Context, realm, service, scope string) (*bearerToken, error) {
	authReq, err := http.NewRequest("GET", realm, nil)
	if err != nil {
		return nil, err
	}
	authReq = authReq.WithContext(ctx)
	getParams := authReq.URL.Query()
	if service != "" {
		getParams.Add("service", service)
	}
	if scope != "" {
		getParams.Add("scope", scope)
	}
	authReq.URL.RawQuery = getParams.Encode()
	if c.username != "" && c.password != "" {
		authReq.SetBasicAuth(c.username, c.password)
	}
	tr := tlsclientconfig.NewTransport()
	// TODO(runcom): insecure for now to contact the external token service
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}
	res, err := client.Do(authReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorizedForCredentials
	case http.StatusOK:
		break
	default:
		return nil, errors.Errorf("unexpected http code: %d, URL: %s", res.StatusCode, authReq.URL)
	}
	tokenBlob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var token bearerToken
	if err := json.Unmarshal(tokenBlob, &token); err != nil {
		return nil, err
	}
	if token.ExpiresIn < minimumTokenLifetimeSeconds {
		token.ExpiresIn = minimumTokenLifetimeSeconds
		logrus.Debugf("Increasing token expiration to: %d seconds", token.ExpiresIn)
	}
	if token.IssuedAt.IsZero() {
		token.IssuedAt = time.Now().UTC()
	}
	return &token, nil
}

// detectProperties detects various properties of the registry.
// See the dockerClient documentation for members which are affected by this.
func (c *dockerClient) detectProperties(ctx context.Context) error {
	if c.scheme != "" {
		return nil
	}

	ping := func(scheme string) error {
		url := fmt.Sprintf(resolvedPingV2URL, scheme, c.registry)
		resp, err := c.makeRequestToResolvedURL(ctx, "GET", url, nil, nil, -1, true)
		logrus.Debugf("Ping %s err %#v", url, err)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		logrus.Debugf("Ping %s status %d", url, resp.StatusCode)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
			return errors.Errorf("error pinging repository, response code %d", resp.StatusCode)
		}
		c.challenges = parseAuthHeader(resp.Header)
		c.scheme = scheme
		c.supportsSignatures = resp.Header.Get("X-Registry-Supports-Signatures") == "1"
		return nil
	}
	err := ping("https")
	if err != nil && c.ctx != nil && c.ctx.DockerInsecureSkipTLSVerify {
		err = ping("http")
	}
	if err != nil {
		err = errors.Wrap(err, "pinging docker registry returned")
		if c.ctx != nil && c.ctx.DockerDisableV1Ping {
			return err
		}
		// best effort to understand if we're talking to a V1 registry
		pingV1 := func(scheme string) bool {
			url := fmt.Sprintf(resolvedPingV1URL, scheme, c.registry)
			resp, err := c.makeRequestToResolvedURL(ctx, "GET", url, nil, nil, -1, true)
			logrus.Debugf("Ping %s err %#v", url, err)
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			logrus.Debugf("Ping %s status %d", url, resp.StatusCode)
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
				return false
			}
			return true
		}
		isV1 := pingV1("https")
		if !isV1 && c.ctx != nil && c.ctx.DockerInsecureSkipTLSVerify {
			isV1 = pingV1("http")
		}
		if isV1 {
			err = ErrV1NotSupported
		}
	}
	return err
}

// getExtensionsSignatures returns signatures from the X-Registry-Supports-Signatures API extension,
// using the original data structures.
func (c *dockerClient) getExtensionsSignatures(ctx context.Context, ref dockerReference, manifestDigest digest.Digest) (*extensionSignatureList, error) {
	path := fmt.Sprintf(extensionsSignaturePath, reference.Path(ref.ref), manifestDigest)
	res, err := c.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, client.HandleErrorResponse(res)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var parsedBody extensionSignatureList
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return nil, errors.Wrapf(err, "Error decoding signature list")
	}
	return &parsedBody, nil
}
