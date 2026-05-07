package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	ServiceName        = "pdfdb"
	DefaultProfileName = "default"
)

type SecretStore interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type KeychainStore struct{}

func (KeychainStore) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (KeychainStore) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (KeychainStore) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type Manager struct {
	path    string
	secrets SecretStore
}

type Config struct {
	ActiveProfile string   `json:"activeProfile"`
	Profiles      []string `json:"profiles"`
}

type Profile struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func NewDefault() (*Manager, error) {
	dir, err := appSupportDir()
	if err != nil {
		return nil, err
	}
	return New(filepath.Join(dir, "profiles.json"), KeychainStore{}), nil
}

func New(path string, secrets SecretStore) *Manager {
	return &Manager{path: path, secrets: secrets}
}

func (m *Manager) List() ([]Profile, error) {
	cfg, err := m.load()
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(cfg.Profiles))
	for _, name := range cfg.Profiles {
		out = append(out, Profile{Name: name, Active: name == cfg.ActiveProfile})
	}
	return out, nil
}

func (m *Manager) Save(name, databaseURL string) error {
	name = cleanName(name)
	if name == "" {
		name = DefaultProfileName
	}
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("database URL is required")
	}
	cfg, err := m.load()
	if err != nil {
		return err
	}
	if !contains(cfg.Profiles, name) {
		cfg.Profiles = append(cfg.Profiles, name)
		sort.Strings(cfg.Profiles)
	}
	cfg.ActiveProfile = name
	if err := m.secrets.Set(ServiceName, account(name), databaseURL); err != nil {
		return fmt.Errorf("save database URL in Keychain: %w", err)
	}
	return m.save(cfg)
}

func (m *Manager) SetActive(name string) error {
	name = cleanName(name)
	cfg, err := m.load()
	if err != nil {
		return err
	}
	if !contains(cfg.Profiles, name) {
		return fmt.Errorf("profile %q does not exist", name)
	}
	if _, err := m.URL(name); err != nil {
		return err
	}
	cfg.ActiveProfile = name
	return m.save(cfg)
}

func (m *Manager) Delete(name string) error {
	name = cleanName(name)
	cfg, err := m.load()
	if err != nil {
		return err
	}
	next := cfg.Profiles[:0]
	for _, profile := range cfg.Profiles {
		if profile != name {
			next = append(next, profile)
		}
	}
	cfg.Profiles = next
	if cfg.ActiveProfile == name {
		cfg.ActiveProfile = ""
		if len(cfg.Profiles) > 0 {
			cfg.ActiveProfile = cfg.Profiles[0]
		}
	}
	_ = m.secrets.Delete(ServiceName, account(name))
	return m.save(cfg)
}

func (m *Manager) ActiveURL() (string, error) {
	cfg, err := m.load()
	if err != nil {
		return "", err
	}
	if cfg.ActiveProfile == "" {
		return "", errors.New("no active database profile configured")
	}
	return m.URL(cfg.ActiveProfile)
}

func (m *Manager) URL(name string) (string, error) {
	url, err := m.secrets.Get(ServiceName, account(cleanName(name)))
	if err != nil {
		return "", fmt.Errorf("read database URL from Keychain: %w", err)
	}
	if strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("profile %q has no database URL", name)
	}
	return url, nil
}

func (m *Manager) Config() (Config, error) {
	return m.load()
}

func (m *Manager) load() (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.ActiveProfile = cleanName(cfg.ActiveProfile)
	cleaned := make([]string, 0, len(cfg.Profiles))
	seen := map[string]bool{}
	for _, name := range cfg.Profiles {
		name = cleanName(name)
		if name != "" && !seen[name] {
			cleaned = append(cleaned, name)
			seen[name] = true
		}
	}
	sort.Strings(cleaned)
	cfg.Profiles = cleaned
	return cfg, nil
}

func (m *Manager) save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, append(data, '\n'), 0o600)
}

func ResolveDatabaseURL() (string, error) {
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value, nil
	}
	if profile := strings.TrimSpace(os.Getenv("PDFDB_PROFILE")); profile != "" {
		mgr, err := NewDefault()
		if err != nil {
			return "", err
		}
		return mgr.URL(profile)
	}
	mgr, err := NewDefault()
	if err == nil {
		if url, err := mgr.ActiveURL(); err == nil && strings.TrimSpace(url) != "" {
			return url, nil
		}
	}
	return "", errors.New("DATABASE_URL is not set and no active pdfdb database profile is configured")
}

func appSupportDir() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "pdfdb"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pdfdb"), nil
}

func account(name string) string {
	return "profile:" + cleanName(name)
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
