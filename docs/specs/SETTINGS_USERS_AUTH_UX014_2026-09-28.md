# SPEC — Settings, Users, Login and Setup (`CCMAI-UX-014`)

**Date:** 2026-09-28 · **Author:** Claude (`ORCHESTRATOR` by owner direction → `SPEC_AUTHOR`; design approved under the owner's design delegation) · **Risk:** R2 · **Work order:** [CCMAI_UX_014](../work_orders/CCMAI_UX_014.md) · **Roadmap:** screen 7. **Status:** BUILD done, `REVIEW_PENDING` ([evidence](../reviews/SETTINGS_USERS_AUTH_UX014_BUILD_2026-09-28.md)).

**Approved design:** canvas https://claude.ai/artifact/8E2fGLG5qEwcmZK6bGVJyS ("Cài đặt & phụ trợ", private to the owner), **version id `1790607141-5a73`**. Artboards: "Cài đặt — 1280 — cấu hình AI", "Cài đặt — 390 dark — lưu trữ file", "Người dùng — 1280 — danh sách", "Đăng nhập, Thiết lập, lỗi tải cài đặt — 390". UX-015 boards will be added to the same canvas under a later version.

## 0. INTAKE (owner-directed)

Presentation of `views/Settings.vue`, `Users.vue`, `Login.vue`, `Setup.vue`. Nothing here changes provider behavior, stored settings, permissions or the auth flow. It does not depend on UX-012/013: the files are disjoint, and the i18n additions are separate `st_*`/`us_*`/`au_*` blocks.

## 1. Fixed contract (source truth at `702c5e3`)

| Fact | Consequence |
|---|---|
| `GET /settings` returns settings (secrets masked as `••••••••`) and tenant fields; `PUT /settings/ai`, `/settings/analysis`, `/settings/general`, `GET/PUT /settings/storage`, `POST /settings/storage/test`, `GET /settings/ai/models`, `POST …/models/refresh`. | Payloads unchanged. |
| When `GET /settings` failed, the old view kept its defaults (Claude, `claude-sonnet-5`, rate 26000, empty company) and every Save sent them. | On failure the forms are hidden behind an alert with "Thử lại"; nothing can be saved until a load succeeds. |
| `POST /settings/ai/test` decrypts the **saved** key and calls the provider. | The button reads "Kiểm tra key đã lưu" with the note "lưu trước nếu vừa đổi". Evidence never presses it (real provider call). |
| Storage save requires a successful connection test (existing rule); turning S3 off asks for confirmation (existing dialog). | Kept. |
| Users: `fetchUsers`, invite (`inviteUser`), role change (`updateRole`, applied on select), permissions PUT, reset password PUT, remove. Backend rejects passwords without 8 characters, an uppercase letter and a digit (`weak_password`). | The create form's rule is aligned with that backend rule (was ≥ 6, so a user could fill a form the server always rejects). Role changes still apply immediately (kept). |
| Member permissions UI offers view/edit (`r`/`rw`); several routes check `d`. | Held: permission contract (§5). |
| Login: 429 message from the backend, 401 invalid credentials; Setup posts name/email/password/workspace. | Behavior unchanged; copy moved to i18n. |

## 2. Intended behavior

1. **Settings:** title; desktop side navigation (4 sections), mobile scrollable tabs. Each section is a card with a short description.
   - AI: provider, model with refresh, the static-list note, API key (masked when saved, with a note), custom URL toggle and field, "Lưu cài đặt", "Kiểm tra key đã lưu" plus its note.
   - Analysis: batch toggle and size.
   - Storage: status note instead of success/info alerts, S3 fields, test result, save gated by the test, the S3-off dialog, and the migration command in a token-coloured code block.
   - General: company, timezone, language, exchange rate, app URL.
   - All Vietnamese copy goes through i18n; snacks use i18n.
2. **Settings load failure:** as §1; a loading skeleton appears before the first load.
3. **Users:** title plus "{n} người dùng"; a table on desktop and cards on mobile. Each shows the name (with "(bạn)" for self), email, a role select with Vietnamese role names (Chủ sở hữu / Quản trị / Thành viên; the same values), "Phân quyền" for members, and a "⋯" menu with "Đặt lại mật khẩu" and "Xóa khỏi công ty" (the existing confirm).
   - Dialogs use `AppDialog` (close button).
   - The create rule is ≥ 8 characters with an uppercase letter and a digit, matching the backend.
   - All snacks are in Vietnamese through i18n.
   - A load error shows an alert with retry.
4. **Login/Setup:** flat bordered card on theme tokens, labels through i18n, `autocomplete` attributes, the password rule shown persistently on Setup, errors as `role="alert"`.
5. Dark mode via tokens.

## 3. Current vs intended

Settings silently rendered defaults on load failure → blocked with retry. "Kiểm tra API key" implied testing the typed key → "Kiểm tra key đã lưu". English snacks ("User created", "Role updated", "Permissions updated") and roles (Owner/Admin/Member) → Vietnamese via i18n. Create-password ≥ 6 vs backend 8+upper+digit → aligned. Icon-only buttons without labels → labelled menu. Hard-coded strings in Setup → i18n.

## 4. Invariants

Request payloads and endpoints unchanged. Role change timing unchanged, and permission values unchanged (`r`/`rw`). No provider test in evidence; no secret shown in evidence.

## 5. Held

1. Delete permission (`d`) for members — permission contract.
2. A role-change confirmation — behavior change, proposed separately.
3. Per-field validation of provider/base URL on the server side — unchanged.

## 6. Acceptance

vue-tsc, build and full vitest. Focused tests with a mocked API (UI only):
- a Settings load failure renders no Save button and no PUT, and retry loads the forms;
- the test button label says the saved key;
- Users renders Vietnamese role names and "(bạn)";
- the create rule rejects `abcdef12` and accepts `Abcdefg1`;
- Users load error + retry;
- Login shows the 401 message.

Disposable captures of Settings (each section), Users, Login and Setup at desktop/mobile × light/dark with 0 JS errors/overflow/external requests; the AI key test is not pressed. Diff/catalog/doctor.
