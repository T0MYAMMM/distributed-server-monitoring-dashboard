// Package settings backs the in-app Settings page: a small set of operator-
// tunable values that were previously reachable only via environment variables
// or the CLI. Precedence is env var > stored override > built-in default, so an
// explicitly-set env var stays authoritative (and is shown locked in the UI),
// while everything else is editable at runtime and persisted in SQLite.
package settings

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"log/slog"
)

// Kind is the value type of a setting, so the UI can render the right control
// and the service can validate input.
type Kind string

const (
	KindString Kind = "string"
	KindInt    Kind = "int"
	KindFloat  Kind = "float"
	KindBool   Kind = "bool"
	KindEnum   Kind = "enum"
)

// Field is one setting plus the metadata the UI needs to render and explain it.
type Field struct {
	Key             string   `json:"key"`
	Section         string   `json:"section"`
	Label           string   `json:"label"`
	Help            string   `json:"help,omitempty"`
	Kind            Kind     `json:"kind"`
	Value           string   `json:"value"`
	Default         string   `json:"default"`
	Options         []string `json:"options,omitempty"`
	EnvVar          string   `json:"env_var,omitempty"`
	EnvLocked       bool     `json:"env_locked"`
	RestartRequired bool     `json:"restart_required,omitempty"`
	Min             *float64 `json:"min,omitempty"`
	Max             *float64 `json:"max,omitempty"`
}

// Doc is the full settings payload returned to the UI: editable fields grouped
// by section plus read-only "About" facts.
type Doc struct {
	Fields []Field           `json:"fields"`
	About  map[string]string `json:"about"`
}

// Setting keys (stable identifiers used by the store and the API).
const (
	KeyInstanceName     = "instance_name"
	KeyDefaultTheme     = "default_theme"
	KeyDefaultDensity   = "default_density"
	KeyTimezone         = "timezone"
	KeyLogRetentionDays = "log_retention_days"
	KeyDiskThreshold    = "alert_disk_threshold"
	KeyStaleAfter       = "stale_after_seconds"
	KeyMaskIPs          = "mask_tailscale_ips"
	KeySessionHours     = "session_hours"
)

func f64(v float64) *float64 { return &v }

// spec is the immutable declaration of one setting: its metadata and default.
// Defaults that mirror an env-backed value are injected at construction so the
// shown default matches the running configuration.
type spec struct {
	field   Field
	envName string
}

// Store persists operator overrides. Defaults and env precedence live here.
type Store interface {
	SettingsAll() (map[string]string, error)
	SetSetting(key, value string) error
}

// Service merges defaults, stored overrides, and env vars into effective
// values, validates updates, and exposes typed live getters used elsewhere in
// the backend. Effective values are cached in memory and refreshed on update.
type Service struct {
	store   Store
	log     *slog.Logger
	specs   []spec
	specMap map[string]spec

	mu     sync.RWMutex
	stored map[string]string
	hooks  []func()
}

// Defaults carries the boot-time defaults (from config/env) the registry needs.
type Defaults struct {
	InstanceName  string
	DiskThreshold float64
	StaleAfter    int
}

// New builds the settings service, loads stored overrides, and primes the
// cache. log may be nil.
func New(store Store, d Defaults, log *slog.Logger) (*Service, error) {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{store: store, log: log, specMap: map[string]spec{}}
	s.specs = registry(d)
	for _, sp := range s.specs {
		s.specMap[sp.field.Key] = sp
	}
	stored, err := store.SettingsAll()
	if err != nil {
		return nil, err
	}
	s.stored = stored
	return s, nil
}

// registry declares every setting once.
func registry(d Defaults) []spec {
	if d.InstanceName == "" {
		d.InstanceName = "CloudGuard"
	}
	return []spec{
		{field: Field{Key: KeyInstanceName, Section: "General", Label: "Instance name", Help: "Shown in the title bar and notifications.", Kind: KindString, Default: d.InstanceName}},
		{field: Field{Key: KeyDefaultTheme, Section: "General", Label: "Default theme", Kind: KindEnum, Options: []string{"dark", "light"}, Default: "dark"}},
		{field: Field{Key: KeyDefaultDensity, Section: "General", Label: "Default density", Kind: KindEnum, Options: []string{"comfortable", "compact"}, Default: "comfortable"}},
		{field: Field{Key: KeyTimezone, Section: "General", Label: "Time zone", Help: "IANA name used to render timestamps (e.g. UTC, Europe/Paris).", Kind: KindString, Default: "UTC"}},

		{field: Field{Key: KeyLogRetentionDays, Section: "Data & retention", Label: "Log retention (days)", Help: "Lines older than this are pruned by the nightly job. 0 keeps everything.", Kind: KindInt, Default: "30", Min: f64(0), Max: f64(3650)}},

		{field: Field{Key: KeyDiskThreshold, Section: "Thresholds", Label: "Disk alert threshold (%)", Help: "Disk usage above this raises a warning alert. 0 disables it.", Kind: KindFloat, Default: strconv.FormatFloat(d.DiskThreshold, 'f', -1, 64), Min: f64(0), Max: f64(100), EnvVar: "ALERT_DISK_THRESHOLD"}, envName: "ALERT_DISK_THRESHOLD"},
		{field: Field{Key: KeyStaleAfter, Section: "Thresholds", Label: "Stale after (seconds)", Help: "Silence before a server is marked stopped.", Kind: KindInt, Default: strconv.Itoa(d.StaleAfter), Min: f64(5), Max: f64(3600), EnvVar: "STALE_AFTER_SECONDS", RestartRequired: true}, envName: "STALE_AFTER_SECONDS"},

		{field: Field{Key: KeyMaskIPs, Section: "Security", Label: "Mask Tailscale & public IPs for anonymous viewers", Help: "When on, addresses are hidden until an admin signs in.", Kind: KindBool, Default: "true"}},
		{field: Field{Key: KeySessionHours, Section: "Security", Label: "Session length (hours)", Help: "How long an admin login stays valid. Applies to new logins.", Kind: KindInt, Default: "24", Min: f64(1), Max: f64(720)}},
	}
}

// effective returns the in-force value for a key: env override if present,
// else stored override, else default.
func (s *Service) effective(sp spec) (value string, envLocked bool) {
	if sp.envName != "" {
		if v := os.Getenv(sp.envName); v != "" {
			return v, true
		}
	}
	s.mu.RLock()
	v, ok := s.stored[sp.field.Key]
	s.mu.RUnlock()
	if ok {
		return v, false
	}
	return sp.field.Default, false
}

// Doc builds the full settings payload. about is merged in by the caller (it
// owns version/build/log-db facts).
func (s *Service) Doc(about map[string]string) Doc {
	fields := make([]Field, 0, len(s.specs))
	for _, sp := range s.specs {
		f := sp.field
		f.Value, f.EnvLocked = s.effective(sp)
		fields = append(fields, f)
	}
	return Doc{Fields: fields, About: about}
}

// Update validates and persists a set of overrides, then refreshes the cache
// and runs apply hooks so live-readable settings take effect immediately.
// Env-locked keys are rejected. Unknown keys are ignored.
func (s *Service) Update(in map[string]string) error {
	clean := map[string]string{}
	for k, v := range in {
		sp, ok := s.specMap[k]
		if !ok {
			continue
		}
		if _, locked := s.effective(sp); locked {
			return fmt.Errorf("%q is set by an environment variable and cannot be changed here", sp.field.Label)
		}
		nv, err := validate(sp, v)
		if err != nil {
			return err
		}
		clean[k] = nv
	}
	for k, v := range clean {
		if err := s.store.SetSetting(k, v); err != nil {
			return err
		}
	}
	s.mu.Lock()
	for k, v := range clean {
		s.stored[k] = v
	}
	hooks := append([]func(){}, s.hooks...)
	s.mu.Unlock()
	for _, fn := range hooks {
		fn()
	}
	return nil
}

// validate checks a value against its spec and normalizes it.
func validate(sp spec, v string) (string, error) {
	switch sp.field.Kind {
	case KindBool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return "", fmt.Errorf("%s must be true or false", sp.field.Label)
		}
		return strconv.FormatBool(b), nil
	case KindInt:
		n, err := strconv.Atoi(v)
		if err != nil {
			return "", fmt.Errorf("%s must be a whole number", sp.field.Label)
		}
		if err := bounds(sp, float64(n)); err != nil {
			return "", err
		}
		return strconv.Itoa(n), nil
	case KindFloat:
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", fmt.Errorf("%s must be a number", sp.field.Label)
		}
		if err := bounds(sp, x); err != nil {
			return "", err
		}
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case KindEnum:
		for _, o := range sp.field.Options {
			if o == v {
				return v, nil
			}
		}
		return "", fmt.Errorf("%s must be one of %v", sp.field.Label, sp.field.Options)
	default:
		return v, nil
	}
}

func bounds(sp spec, x float64) error {
	if sp.field.Min != nil && x < *sp.field.Min {
		return fmt.Errorf("%s must be at least %v", sp.field.Label, *sp.field.Min)
	}
	if sp.field.Max != nil && x > *sp.field.Max {
		return fmt.Errorf("%s must be at most %v", sp.field.Label, *sp.field.Max)
	}
	return nil
}

// OnApply registers a callback run after every successful update (and once at
// startup via ApplyNow) so callers can push live-readable settings into other
// services.
func (s *Service) OnApply(fn func()) {
	s.mu.Lock()
	s.hooks = append(s.hooks, fn)
	s.mu.Unlock()
}

// ApplyNow runs the registered hooks immediately, so boot-time stored overrides
// reach the services that consume them.
func (s *Service) ApplyNow() {
	s.mu.RLock()
	hooks := append([]func(){}, s.hooks...)
	s.mu.RUnlock()
	for _, fn := range hooks {
		fn()
	}
}

// --- typed live getters ---

func (s *Service) str(key string) string {
	v, _ := s.effective(s.specMap[key])
	return v
}

// InstanceName returns the configured instance name.
func (s *Service) InstanceName() string { return s.str(KeyInstanceName) }

// MaskIPs reports whether IP addresses should be masked for anonymous viewers.
func (s *Service) MaskIPs() bool {
	b, err := strconv.ParseBool(s.str(KeyMaskIPs))
	if err != nil {
		return true
	}
	return b
}

// DiskThreshold returns the effective disk alert threshold percent.
func (s *Service) DiskThreshold() float64 {
	x, err := strconv.ParseFloat(s.str(KeyDiskThreshold), 64)
	if err != nil {
		return 90
	}
	return x
}

// LogRetentionDays returns the configured log retention window in days (0 = keep all).
func (s *Service) LogRetentionDays() int {
	n, err := strconv.Atoi(s.str(KeyLogRetentionDays))
	if err != nil {
		return 30
	}
	return n
}

// StaleAfterSeconds returns the silence window before a server is marked
// stopped. Read once at boot (the sweeper interval is fixed thereafter).
func (s *Service) StaleAfterSeconds() int {
	n, err := strconv.Atoi(s.str(KeyStaleAfter))
	if err != nil || n <= 0 {
		return 30
	}
	return n
}

// SessionHours returns how long a new admin session stays valid.
func (s *Service) SessionHours() int {
	n, err := strconv.Atoi(s.str(KeySessionHours))
	if err != nil || n <= 0 {
		return 24
	}
	return n
}
