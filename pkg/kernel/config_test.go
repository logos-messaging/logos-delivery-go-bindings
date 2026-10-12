package kernel

import (
	"encoding/json"
	"testing"
)

// The library still accepts a legacy flat configuration blob and tells the two
// shapes apart by looking for bare top-level keys, so an empty Config must
// marshal to an empty object rather than to a set of zero values.
func TestConfigOmitsEmptyFields(t *testing.T) {
	got, err := json.Marshal(Config{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("Marshal(Config{}) = %s, want {}", got)
	}
}

func TestConfigMarshalsLayeredShape(t *testing.T) {
	got, err := json.Marshal(Config{
		Mode:               ModeCore,
		Preset:             PresetLogosDev,
		MessagingOverrides: Overrides{"tcp-port": 60000},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"mode":"Core","preset":"logos.dev","messagingOverrides":{"tcp-port":60000}}`
	if string(got) != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}
