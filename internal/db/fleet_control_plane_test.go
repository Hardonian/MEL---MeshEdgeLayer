package db

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func testFleetDB(t *testing.T) *DB {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	seed := `
INSERT INTO organizations(id, slug, name) VALUES ('org1','org1','Org 1');
INSERT INTO workspaces(id, organization_id, slug, name) VALUES ('ws1','org1','alpha','Alpha');
INSERT INTO workspaces(id, organization_id, slug, name) VALUES ('ws2','org1','bravo','Bravo');
INSERT INTO workspace_memberships(workspace_id, actor_id, role) VALUES ('ws1','actor:owner','owner');
INSERT INTO workspace_memberships(workspace_id, actor_id, role) VALUES ('ws1','actor:viewer','viewer');
INSERT INTO workspace_memberships(workspace_id, actor_id, role) VALUES ('ws2','actor:owner','owner');
`
	if err := d.ExecScript(seed); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDeviceRegistryConflictAndRelink(t *testing.T) {
	d := testFleetDB(t)
	if conflict, err := d.UpsertDeviceIdentity(DeviceIdentityInput{WorkspaceID: "ws1", DeviceID: "dev-1", CanonicalKey: "canon-1", DisplayName: "Node A", IdentityType: "hardware_id", IdentityValue: "hw-1", ActorID: "actor:owner"}); err != nil || conflict != nil {
		t.Fatalf("initial upsert err=%v conflict=%+v", err, conflict)
	}
	conflict, err := d.UpsertDeviceIdentity(DeviceIdentityInput{WorkspaceID: "ws1", DeviceID: "dev-2", CanonicalKey: "canon-2", DisplayName: "Node B", IdentityType: "hardware_id", IdentityValue: "hw-1", ActorID: "actor:owner"})
	if err != nil {
		t.Fatal(err)
	}
	if conflict == nil || conflict.State != "identity_conflict" {
		t.Fatalf("expected identity_conflict, got %+v", conflict)
	}
	conflict, err = d.UpsertDeviceIdentity(DeviceIdentityInput{WorkspaceID: "ws1", DeviceID: "dev-2", CanonicalKey: "canon-2", DisplayName: "Node B", IdentityType: "hardware_id", IdentityValue: "hw-1", ActorID: "actor:owner", AllowRelink: true})
	if err != nil || conflict != nil {
		t.Fatalf("relink err=%v conflict=%+v", err, conflict)
	}
	rows, err := d.QueryRows("SELECT COUNT(*) AS c FROM device_alias_history WHERE workspace_id='ws1' AND device_id='dev-2';")
	if err != nil {
		t.Fatal(err)
	}
	if asInt(rows[0]["c"]) == 0 {
		t.Fatalf("expected alias history for relinked device")
	}
}

func TestRolloutStateTransitionsAndTenantIsolation(t *testing.T) {
	d := testFleetDB(t)
	_, _ = d.UpsertDeviceIdentity(DeviceIdentityInput{WorkspaceID: "ws1", DeviceID: "dev-1", CanonicalKey: "canon-1", ActorID: "actor:owner"})
	_, _ = d.UpsertDeviceIdentity(DeviceIdentityInput{WorkspaceID: "ws1", DeviceID: "dev-2", CanonicalKey: "canon-2", ActorID: "actor:owner"})
	if err := d.CreateRolloutJob("ws1", "actor:owner", "job-1", "", "apply_template", "selected_devices", "", []RolloutTargetInput{{TargetID: "t1", DeviceID: "dev-1"}, {TargetID: "t2", DeviceID: "dev-2", Offline: true}}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpdateRolloutTargetState("ws1", "actor:owner", "t1", "acknowledged", "", false); err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryRows("SELECT state FROM rollout_jobs WHERE id='job-1';")
	if err != nil {
		t.Fatal(err)
	}
	if got := asString(rows[0]["state"]); got != "partial_failure" {
		t.Fatalf("rollout state=%s want partial_failure", got)
	}
	if _, err := d.ListFleetDashboard("ws2", "actor:viewer", 10, 0); err == nil {
		t.Fatal("expected tenant isolation membership error")
	}
}

func TestAlertTriggerAndAcknowledge(t *testing.T) {
	d := testFleetDB(t)
	if _, err := d.UpsertDeviceIdentity(DeviceIdentityInput{WorkspaceID: "ws1", DeviceID: "dev-1", CanonicalKey: "canon-1", ActorID: "actor:owner"}); err != nil {
		t.Fatal(err)
	}
	if err := d.TriggerAlert("ws1", "actor:owner", "alert-1", "dev-1", "offline_too_long", "critical", "Device offline", "No heartbeat"); err != nil {
		t.Fatal(err)
	}
	if err := d.AcknowledgeAlert("ws1", "actor:owner", "alert-1"); err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryRows("SELECT state, acknowledged_by FROM alerts WHERE id='alert-1';")
	if err != nil {
		t.Fatal(err)
	}
	if asString(rows[0]["state"]) != "acknowledged" || asString(rows[0]["acknowledged_by"]) != "actor:owner" {
		t.Fatalf("unexpected alert row: %+v", rows[0])
	}
}
