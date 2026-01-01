package featureflags

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/api"
	"github.com/Unleash/unleash-client-go/v4/context"
)

type Config struct {
	URL             string
	AppName         string
	APIToken        string
	Environment     string
	RefreshInterval time.Duration
	MetricsInterval time.Duration
	DisableMetrics  bool
}

type Context struct {
	UserID     string
	TenantID   string
	SessionID  string
	RemoteAddr string
	Properties map[string]string
}

func NewContext() *Context {
	return &Context{
		Properties: make(map[string]string),
	}
}

func (c *Context) WithUserID(userID string) *Context {
	c.UserID = userID
	return c
}

func (c *Context) WithTenantID(tenantID string) *Context {
	c.TenantID = tenantID
	return c
}

func (c *Context) WithSessionID(sessionID string) *Context {
	c.SessionID = sessionID
	return c
}

func (c *Context) WithRemoteAddr(remoteAddr string) *Context {
	c.RemoteAddr = remoteAddr
	return c
}

func (c *Context) WithProperty(key, value string) *Context {
	c.Properties[key] = value
	return c
}

func (c *Context) toUnleashContext() context.Context {
	ctx := context.Context{}
	if c == nil {
		return ctx
	}

	ctx.UserId = c.UserID
	ctx.SessionId = c.SessionID
	ctx.RemoteAddress = c.RemoteAddr

	if c.TenantID != "" {
		if ctx.Properties == nil {
			ctx.Properties = make(map[string]string)
		}
		ctx.Properties["tenantId"] = c.TenantID
	}

	for k, v := range c.Properties {
		if ctx.Properties == nil {
			ctx.Properties = make(map[string]string)
		}
		ctx.Properties[k] = v
	}

	return ctx
}

type Client struct {
	config  Config
	ready   bool
	readyMu sync.RWMutex
}

var (
	once     sync.Once
	instance *Client
)

func New(cfg Config) (*Client, error) {
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 15 * time.Second
	}
	if cfg.MetricsInterval == 0 {
		cfg.MetricsInterval = 60 * time.Second
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	client := &Client{
		config: cfg,
		ready:  false,
	}

	headers := http.Header{}
	headers.Set("Authorization", cfg.APIToken)

	err := unleash.Initialize(
		unleash.WithUrl(cfg.URL),
		unleash.WithAppName(cfg.AppName),
		unleash.WithCustomHeaders(headers),
		unleash.WithRefreshInterval(cfg.RefreshInterval),
		unleash.WithMetricsInterval(cfg.MetricsInterval),
		unleash.WithDisableMetrics(cfg.DisableMetrics),
		unleash.WithListener(&listener{client: client}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize unleash: %w", err)
	}

	return client, nil
}

func GetInstance() *Client {
	return instance
}

func InitGlobal(cfg Config) error {
	var initErr error
	once.Do(func() {
		var err error
		instance, err = New(cfg)
		if err != nil {
			initErr = err
		}
	})
	return initErr
}

func (c *Client) WaitForReady(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.readyMu.RLock()
		ready := c.ready
		c.readyMu.RUnlock()
		if ready {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func (c *Client) IsReady() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.ready
}

func (c *Client) Close() {
	unleash.Close()
}

func (c *Client) IsGlobalEnabled(featureName string) bool {
	fullName := fmt.Sprintf("global.%s.%s", c.config.AppName, featureName)
	return unleash.IsEnabled(fullName)
}

func (c *Client) IsGlobalEnabledWithFallback(featureName string, fallback bool) bool {
	fullName := fmt.Sprintf("global.%s.%s", c.config.AppName, featureName)
	return unleash.IsEnabled(fullName, unleash.WithFallback(fallback))
}

func (c *Client) IsTenantEnabled(featureName string, ctx *Context) bool {
	if ctx == nil || ctx.TenantID == "" {
		return false
	}
	fullName := fmt.Sprintf("tenant.%s.%s", c.config.AppName, featureName)
	return unleash.IsEnabled(fullName, unleash.WithContext(ctx.toUnleashContext()))
}

func (c *Client) IsTenantEnabledWithFallback(featureName string, ctx *Context, fallback bool) bool {
	if ctx == nil || ctx.TenantID == "" {
		return fallback
	}
	fullName := fmt.Sprintf("tenant.%s.%s", c.config.AppName, featureName)
	return unleash.IsEnabled(fullName, unleash.WithContext(ctx.toUnleashContext()), unleash.WithFallback(fallback))
}

func (c *Client) IsUserEnabled(featureName string, ctx *Context) bool {
	if ctx == nil || ctx.UserID == "" {
		return false
	}
	fullName := fmt.Sprintf("user.%s.%s", c.config.AppName, featureName)
	return unleash.IsEnabled(fullName, unleash.WithContext(ctx.toUnleashContext()))
}

func (c *Client) IsUserEnabledWithFallback(featureName string, ctx *Context, fallback bool) bool {
	if ctx == nil || ctx.UserID == "" {
		return fallback
	}
	fullName := fmt.Sprintf("user.%s.%s", c.config.AppName, featureName)
	return unleash.IsEnabled(fullName, unleash.WithContext(ctx.toUnleashContext()), unleash.WithFallback(fallback))
}

func (c *Client) IsEnabled(featureName string, ctx *Context) bool {
	if ctx == nil {
		return unleash.IsEnabled(featureName)
	}
	return unleash.IsEnabled(featureName, unleash.WithContext(ctx.toUnleashContext()))
}

func (c *Client) IsEnabledWithFallback(featureName string, ctx *Context, fallback bool) bool {
	opts := []unleash.FeatureOption{unleash.WithFallback(fallback)}
	if ctx != nil {
		opts = append(opts, unleash.WithContext(ctx.toUnleashContext()))
	}
	return unleash.IsEnabled(featureName, opts...)
}

func (c *Client) GetVariant(featureName string, ctx *Context) *api.Variant {
	var variant *api.Variant
	if ctx == nil {
		variant = unleash.GetVariant(featureName)
	} else {
		variant = unleash.GetVariant(featureName, unleash.WithVariantContext(ctx.toUnleashContext()))
	}
	return variant
}

type listener struct {
	client *Client
}

func (l *listener) OnReady() {
	l.client.readyMu.Lock()
	l.client.ready = true
	l.client.readyMu.Unlock()
}

func (l *listener) OnError(err error) {}

func (l *listener) OnWarning(warning error) {}

func (l *listener) OnCount(featureName string, enabled bool) {}

func (l *listener) OnSent(payload unleash.MetricsData) {}

func (l *listener) OnRegistered(payload unleash.ClientData) {}
