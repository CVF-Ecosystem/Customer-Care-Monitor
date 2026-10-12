# Package A - auth, dashboard permission projection and refresh CAS

Memory class: PLANNING_RECORD
docType: specification
Status: PARKED
Risk: R2
Author: Codex /root
Date: 2026-10-13

## Locked behavior and finite outcome

Only inheritance F01/F02/F03. Application security semantics, not CVF AI governance or full stage acceptance. Worker /root/auth_intake, independent reviewer Codex /root; current user authorizes Luna max workers. Public Core read-only. Existing R088 accepted; accounts/deploy/product debt remain separate.

A01: Password reset preserves existing tenant actor policy: owner may reset member/admin/owner; admin may reset members only, never admin/owner. Denied/nonmember targets produce existing appropriate HTTP denial/not-found and no password/token_version update. Validate target membership/role and perform password/version change in a transaction with row locking or equivalent conditional race-safe mutation; concurrent promotion must not allow reset of a protected target. Existing role-update rule is authority. Token revocation increments atomically. No broader cross-tenant account redesign.

A02: Dashboard response is projected by existing PermissionDenial(role, permissionJSON, resource, r): owner/admin retain all fields; members get only fields they may read. channels:r -> active_channels; messages:r -> total_conversations, conversations_by_channel, messages_by_day; jobs:r -> active_jobs, issues, qc_violation_count, qc_alerts, classification_recent; settings:r -> cost_period, cost_today, cost_this_month, cost_by_day, exchange_rate. Omit unauthorized keys; do not label missing data as zero. Execute only applicable category queries; no read permission ->403 generic permission denial. Malformed/empty/unknown role fails closed. Keep existing authorized query/date/business-day/error semantics. UI hides unavailable sections and notification-log nav, and tenant changes cannot retain prior privileged dashboard data. No broad layout/style change or demo/import/reset scope.

A03: Notification-log REST route requires settings:r, matching existing MCP policy and other server log routes. Align DefaultLayout menu from jobs to settings. Default members channels/messages/jobs:r lose notification-log access intentionally; jobs:r alone must not expose raw output/cost/evidence through that route. Owner/admin/settings:r controls succeed. MCP policy unchanged.

A04: Refresh cookie token version is a single-winner CAS: UPDATE user WHERE id AND expected token_version, expression increment or expected+1 after validation, require no error AND exactly one affected row before minting tokens/setting successor cookie. Zero row rejects revoked/race-lost token and cannot overwrite a reset increment; DB error gives failure and no successful token/cookie. Preserve HttpOnly cookie transport and existing token claims/login semantics. Token-generation failure may revoke old token but must not publish successful credentials; no rollback to a previously valid version.

## Mandatory discriminating proof

Use actual registered Gin/middleware/handler paths where practical, real GORM against a fresh owned synthetic MySQL fixture (no mandatory skip), and real mounted Dashboard/layout with API boundary mocked. Direct helper-only checks cannot close REST/persistence semantics. Before production edits, tests on the old source must assert admin->owner/admin denial and unchanged persistence; unique forbidden dashboard/log payload markers absent; concurrent same-cookie one winner and stale refresh after reset cannot downgrade version. GORM test callbacks may coordinate reads/updates and force failure without production-specific hooks. Missing helper/export/setup failure is not detector credit. Healthy owner/admin/member allowed controls pass. Cover permission combinations, malformed data, update-error/zero-row, race and current authorized dashboard regressions.

Force relevant handler/router compile/test plus existing dashboard business-date/QC tests with documented fixture identity corrections only (explicit owner for old identity-free direct-handler fixtures), then frontend focused mounted tests and forced type/build once. Preserve original assertions. Add source-bound exact commands/counts/skips/hashes and every failed invocation. No optional broad mutation campaign; one source-negative then one final campaign, repeat only named actual changes/failures. One consolidated repair set; third-round escalation. Stop after these three findings plus Local independent review, then package C.

## Source and authority

Immutable seed CVF_SESSION/authority/CCMAI-RUNTIME-089.json at 7d7b6e483973faf79ead2240728b16f4f0597376; source audit ../probes/r089_readiness/source-audit.json. Allowed paths/effects exactly seed; package/go sum/config/Core/gates protected. Public checksum-pinned Go archive already prepared by Local in ignored runtime; module download and one fresh mysql image/fixture admitted by seed. SQL uses synthetic fixture only, never existing/production DB. No provider/customer/account/real outbound effects.

## Checker Source Read-Ahead Block

Existing legacy authority first-commit/base/role/path/status grammar and public profile active-status/graph/role/routing/claim/continuity/catalog readers govern. New packet uses opt-in workOrderStatusGrammar and active BUILD/APPL actually checked before edit. Test authority derives local policy sources, not provider memory. Bounded source intake only.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. Preparation before actual project branch publication; no private Core export or hosted/live/deploy claim.

Historical unexecuted invalid R089 preparation; current valid SPEC docs/reviews/planning/R090_AUTH_PERMISSIONS_SPEC_2026-10-13.md.
