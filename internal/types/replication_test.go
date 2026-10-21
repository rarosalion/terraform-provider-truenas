package types

import (
	"encoding/json"
	"testing"
)

func TestReplicationUnmarshalSSHCredentials(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"integer from create/update", `{"id":2,"ssh_credentials":5}`, 5},
		{"expanded object from get/query", `{"id":2,"ssh_credentials":{"id":5,"name":"router01","type":"SSH_CREDENTIALS"}}`, 5},
		{"null", `{"id":2,"ssh_credentials":null}`, 0},
		{"absent", `{"id":2}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r Replication
			if err := json.Unmarshal([]byte(tt.body), &r); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if r.SSHCredentials != tt.want {
				t.Errorf("SSHCredentials = %d, want %d", r.SSHCredentials, tt.want)
			}
		})
	}
}

func TestReplicationUnmarshalPullTask(t *testing.T) {
	body := `{
		"id": 2, "name": "router01", "direction": "PULL", "transport": "SSH",
		"source_datasets": ["zroot/ROOT/default"], "target_dataset": "files/backups/router01",
		"recursive": true, "auto": true, "enabled": true, "retention_policy": "SOURCE",
		"replicate": true, "properties": true, "properties_exclude": [], "properties_override": {},
		"name_regex": ".*", "readonly": "SET", "allow_from_scratch": true,
		"hold_pending_snapshots": false, "retries": 5, "large_block": true,
		"compressed": true, "embed": false, "speed_limit": null, "logging_level": null,
		"sudo": false, "exclude": [], "naming_schema": null,
		"schedule": {"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*", "begin": "00:00", "end": "23:59"},
		"ssh_credentials": {"id": 5, "type": "SSH_CREDENTIALS", "attributes": {"username": "truenas"}}
	}`
	var r Replication
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.SSHCredentials != 5 || !r.Replicate || r.NameRegex != ".*" || r.Readonly != "SET" || r.Retries != 5 {
		t.Errorf("unexpected decode: %+v", r)
	}
	if r.Schedule == nil || r.Schedule.Hour != "2" || r.Schedule.End != "23:59" {
		t.Errorf("unexpected schedule: %+v", r.Schedule)
	}
	if r.SpeedLimit != nil || r.LoggingLevel != nil {
		t.Errorf("null speed_limit/logging_level should decode to nil")
	}
}

func TestReplicationCreateRequestOmitsUnsetOptions(t *testing.T) {
	b, err := json.Marshal(&ReplicationCreateRequest{Name: "x", Direction: "PUSH", Transport: "LOCAL"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"replicate", "properties", "retries", "large_block", "schedule", "speed_limit", "sudo", "readonly"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s should be omitted when unset", k)
		}
	}
}
