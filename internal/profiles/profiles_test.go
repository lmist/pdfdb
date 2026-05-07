package profiles

import (
	"path/filepath"
	"testing"
)

type memorySecrets map[string]string

func (m memorySecrets) Get(service, user string) (string, error) {
	return m[service+"\x00"+user], nil
}

func (m memorySecrets) Set(service, user, password string) error {
	m[service+"\x00"+user] = password
	return nil
}

func (m memorySecrets) Delete(service, user string) error {
	delete(m, service+"\x00"+user)
	return nil
}

func TestProfilesSaveAndActiveURL(t *testing.T) {
	t.Parallel()
	mgr := New(filepath.Join(t.TempDir(), "profiles.json"), memorySecrets{})
	if err := mgr.Save("work", "postgres://example"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Save("other", "postgres://other"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.SetActive("work"); err != nil {
		t.Fatal(err)
	}
	url, err := mgr.ActiveURL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "postgres://example" {
		t.Fatalf("url = %q", url)
	}
	profiles, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || !profiles[1].Active || profiles[1].Name != "work" {
		t.Fatalf("profiles = %#v", profiles)
	}
}

func TestProfilesDeleteMovesActive(t *testing.T) {
	t.Parallel()
	mgr := New(filepath.Join(t.TempDir(), "profiles.json"), memorySecrets{})
	if err := mgr.Save("a", "postgres://a"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Save("b", "postgres://b"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Delete("b"); err != nil {
		t.Fatal(err)
	}
	cfg, err := mgr.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProfile != "a" {
		t.Fatalf("active = %q", cfg.ActiveProfile)
	}
}
