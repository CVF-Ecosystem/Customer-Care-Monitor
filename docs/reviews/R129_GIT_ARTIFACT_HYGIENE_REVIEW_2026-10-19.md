# R129 repository artifact hygiene review

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Codex /root repository artifact hygiene
Reviewer: Owner independent review pending

The 72 MB R126 ZIP contains raw local verification logs and test outputs, not application source. It expands to 1,600,128,989 bytes. The archive was moved intact to the workspace-level local evidence backup; the branch will ignore its project path and omit it from the current tree. Catalog, index, and R126 review references were updated to reflect that location and publication boundary.

The change does not merge into main or rewrite branch history. Existing commits continue to contain the archive blob. Independent owner review and tranche closure remain pending.
