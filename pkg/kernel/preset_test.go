package kernel

import (
	"testing"
)

const testStoreNode = "/ip4/127.0.0.1/tcp/60001/p2p/16Uiu2HAm3xVDaz6SRJ6kErwC21zBJEZjavVXg7VSkoWzaV1aMA3F"

func TestAddPresetReachesNewNodes(t *testing.T) {
	err := AddPreset("bindings.test-preset", PresetFields{
		"base":           PresetStatusStaging,
		"entryNodes":     []string{},
		"storeNodes":     []string{testStoreNode},
		"maxMessageSize": "200KiB",
	})
	if err != nil {
		t.Fatalf("AddPreset: %v", err)
	}

	node, err := New(Config{Preset: "bindings.test-preset"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer node.Close()

	storeNodes, err := node.Debug().StoreNodes()
	if err != nil {
		t.Fatalf("StoreNodes: %v", err)
	}
	if len(storeNodes) != 1 || storeNodes[0].String() != testStoreNode {
		t.Errorf("StoreNodes() = %v, want [%s]", storeNodes, testStoreNode)
	}

	size, err := node.Debug().MaxMessageSize()
	if err != nil {
		t.Fatalf("MaxMessageSize: %v", err)
	}
	if size != 200*1024 {
		t.Errorf("MaxMessageSize() = %d, want %d", size, 200*1024)
	}
}

func TestAddPresetRejectsInvalidPreset(t *testing.T) {
	cases := map[string]PresetFields{
		"unknown base":     {"base": "no.such.preset"},
		"unknown field":    {"base": PresetStatusStaging, "colour": "red"},
		"no base, no id":   {"entryNodes": []string{}},
		"wrong field type": {"base": PresetStatusStaging, "clusterId": "sixteen"},
	}
	for name, fields := range cases {
		t.Run(name, func(t *testing.T) {
			if err := AddPreset("bindings.invalid", fields); err == nil {
				t.Errorf("AddPreset(%v) = nil, want an error", fields)
			}
		})
	}
}
