-- Migration: 0022_control_action_policy_truth.sql
-- Description: Persist explicit single-approver policy basis and execution source for control actions.
-- Created: 2026-03-26

ALTER TABLE control_actions ADD COLUMN approval_mode TEXT NOT NULL DEFAULT 'single_approver';
ALTER TABLE control_actions ADD COLUMN required_approvals INTEGER NOT NULL DEFAULT 1;
ALTER TABLE control_actions ADD COLUMN collected_approvals INTEGER NOT NULL DEFAULT 0;
ALTER TABLE control_actions ADD COLUMN approval_basis_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE control_actions ADD COLUMN approval_policy_source TEXT NOT NULL DEFAULT 'mel_config.control';
ALTER TABLE control_actions ADD COLUMN high_blast_radius INTEGER NOT NULL DEFAULT 0;
ALTER TABLE control_actions ADD COLUMN approval_escalated_due_to_blast_radius INTEGER NOT NULL DEFAULT 0;
ALTER TABLE control_actions ADD COLUMN execution_source TEXT NOT NULL DEFAULT '';
