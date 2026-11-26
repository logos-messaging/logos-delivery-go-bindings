package kernel

import (
	"testing"
)

const testStoreNode = "/ip4/127.0.0.1/tcp/60001/p2p/16Uiu2HAm3xVDaz6SRJ6kErwC21zBJEZjavVXg7VSkoWzaV1aMA3F"

func testPreset() PresetFields {
	return PresetFields{
		"clusterId":      16,
		"maxMessageSize": "200KiB",
		"shardingConf":   []uint16{32},
		"storeNodes":     []string{testStoreNode},
	}
}

func TestAddPresetReachesNewNodes(t *testing.T) {
	if err := AddPreset("bindings.test-preset", testPreset()); err != nil {
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
	without := func(key string) PresetFields {
		p := testPreset()
		delete(p, key)
		return p
	}
	with := func(key string, value any) PresetFields {
		p := testPreset()
		p[key] = value
		return p
	}
	cases := map[string]struct {
		name   string
		fields PresetFields
	}{
		"built-in name":    {PresetStatusStaging, testPreset()},
		"unknown field":    {"bindings.invalid", with("colour", "red")},
		"no sharding":      {"bindings.invalid", without("shardingConf")},
		"wrong field type": {"bindings.invalid", with("clusterId", "sixteen")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if err := AddPreset(c.name, c.fields); err == nil {
				t.Errorf("AddPreset(%q, %v) = nil, want an error", c.name, c.fields)
			}
		})
	}
}
