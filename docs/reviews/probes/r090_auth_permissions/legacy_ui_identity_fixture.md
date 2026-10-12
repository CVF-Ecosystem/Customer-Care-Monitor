# Legacy UI fixture identity attempt

Attribution: `/root/auth_intake`, R090 implementation worker. This is a preserved proposed fixture-only patch, not applied source state. Root review identified both fixture paths as outside the immutable R090 seed `allowedPaths`; both live files were restored byte-for-byte to HEAD before production edits. Assertions were unchanged. The owner identity was needed because the new Dashboard permission gate reads tenant permissions, while the older direct-mount tests had no authenticated tenant fixture.

| Path | HEAD Git blob | Restored file SHA-256 |
|---|---|---|
| `frontend/src/__tests__/dashboard-qc-card.spec.ts` | `e9bbfd227923cf5358c4db351f3bf621365287e8` | `8a710fdcc0f127c10440abdc4e8ff12a0b538a3b3c2dbfc658cfe9a2388c024f` |
| `frontend/src/__tests__/dashboard-business-day.spec.ts` | `731fecfd4b91bff011c0a56708509a3a5f364ca7` | `5e5e65dd4faf6c39c8dcf84b6e5f5bfff6a621b8675a6c07ecb48ade0f6540b4` |

The raw candidate hunks are in `legacy_ui_identity_fixture.diff`. Root plans to admit these two fixture-only changes in the already planned C seed. No changes to either legacy test path remain in this R090 tree.
