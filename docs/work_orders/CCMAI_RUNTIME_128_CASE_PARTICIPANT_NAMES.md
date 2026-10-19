# Case-scoped participant names for read-only reviewers

Status: REVIEWER_ACCEPTED
Risk: R2
Worker: Sol Medium backend participant scope; Luna Max frontend reader labels
Reviewer: Root independent source/native/browser evidence review

Observed R127-A browser issue: reader can see current/history cases but all participant names fall back because the UI only loads the editor assignee list. Add optional detail participants containing only user_id/name for same-tenant users referenced by that case, including historical assignment/data ownership. Keep full directory separate and require both read/write permissions there. Do not grant reader actions or expose unrelated users, emails or roles.

Exit: native reader detail names and privacy/cross-tenant/error checks, directory403 for reader and200 for authorized editor; UI renders current/history names with no action controls or full-directory call, preserving route/session lifetime; relevant UI tests/build and current browser proof with original terminal. Reuse existing controlled synthetic fixture, no paid calls/schema changes or installed app mutation. Root independently reviews both workers. No unrelated polish/framework or full-suite rerun without a concrete source dependency concern.
