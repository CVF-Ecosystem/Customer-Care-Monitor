#!/usr/bin/env python3
"""R088 optional mutation proof: each mutation is applied to a disposable overlay copy (never the live
worktree) with an exact expected match count, the four new lifetime suites run, and the result is
classified KILLED (a named behavioral assertion failed), SURVIVED (all passed), NOT_APPLIED (match count
differs) or INCONCLUSIVE (suite-level/setup/timeout failure with no failing assertion).
"""
import json
import os
import subprocess
import sys

PROJECT, WORK, OUT = sys.argv[1], sys.argv[2], sys.argv[3]
RUNNER = os.path.join(PROJECT, "docs/reviews/probes/r088_messages_lifetime/r088_overlay_runner.py")
FILES = ["src/__tests__/conversations-lifetime.spec.ts", "src/__tests__/messages-lifetime.spec.ts",
         "src/__tests__/messages-lifetime-startup.spec.ts", "src/__tests__/messages-lifetime-resources.spec.ts"]
V = "frontend/src/views/Messages.vue"
S = "frontend/src/stores/conversations.ts"

M = [
    ("M01", "store: detail generation check removed", S,
     "      if (generation !== detailGeneration) return false\n      messages.value = data.messages", "      messages.value = data.messages"),
    ("M02", "store: list generation check removed", S,
     "      if (generation !== listGeneration) return false\n      conversations.value = data.data", "      conversations.value = data.data"),
    ("M03", "store: superseded detail failure is rethrown", S,
     "      if (generation !== detailGeneration) return false\n      throw error", "      throw error"),
    ("M04", "store: detail data is not cleared when a new detail call starts", S,
     "    messages.value = []\n    currentConversation.value = null\n    try {\n      const { data } = await api.get(`/tenants/${tenantId}/conversations/${conversationId}/messages`)",
     "    try {\n      const { data } = await api.get(`/tenants/${tenantId}/conversations/${conversationId}/messages`)"),
    ("M05", "store: owner gating removed from resetDetail", S,
     "    if (owner !== undefined && owner !== detailOwner) return\n", ""),
    ("M06", "store: owner gating removed from the list part of reset", S,
     "    if (owner === undefined || owner === listOwner) {", "    if (true) {"),
    ("M07", "component: continuation guard after fetchMessages removed", V,
     "    published = await conversationStore.fetchMessages(life.tenant, convId, owner)\n    if (!isCurrent()) return\n", "    published = await conversationStore.fetchMessages(life.tenant, convId, owner)\n"),
    ("M08", "component: stale finally clears the current messages spinner", V,
     "    if (isCurrent()) loadingMessages.value = false", "    loadingMessages.value = false"),
    ("M09", "component: stale evaluation success publishes", V,
     "    if (!isCurrent()) return\n    evaluation.value = data", "    evaluation.value = data"),
    ("M10", "component: selection generation not advanced on re-selection (A -> B -> A by id only)", V,
     "  const gen = ++selectionGen // every explicit selection", "  const gen = selectionGen // every explicit selection"),
    ("M11", "component: tenant watcher disabled", V,
     "  if (next === previous) return\n  endLifetime()", "  return\n  endLifetime()"),
    ("M12", "component: initialization continues after the channel step", V,
     "  if (!alive()) return\n  await loadConversations()", "  await loadConversations()"),
    ("M13", "component: deep-link continuation after the page lookup", V,
     "      if (!alive()) return\n      if (data?.page", "      if (data?.page"),
    ("M14", "component: object URL allocated for a finished lifetime", V,
     "      if (epoch !== imageEpoch) return\n      authImageCache.value[url] = URL.createObjectURL(blob)", "      authImageCache.value[url] = URL.createObjectURL(blob)"),
    ("M15", "component: invalidation keeps 'loading' markers", V,
     "    if (value === 'loading') delete authImageCache.value[key]", "    void value"),
    ("M16", "component: stale list catch sets the list error", V,
     "    if (isCurrent()) listError.value = true", "    listError.value = true"),
    ("M17", "component: deselect does not reset the store detail", V,
     "  conversationStore.resetDetail(owner)\n}", "}"),
    ("M18", "component: releaseImages does not revoke", V,
     "    if (blobUrl && blobUrl.startsWith('blob:')) URL.revokeObjectURL(blobUrl)\n  }\n  authImageCache.value = {}", "  }\n  authImageCache.value = {}"),
    ("M19", "component: status map applies without lifetime check", V,
     "    if (!isCurrent()) return\n    evaluationMap.value = data || {}", "    evaluationMap.value = data || {}"),
    ("M20", "component: header may show the store's non-matching conversation (defense in depth)", V,
     "  return c && c.id === selectedConvId.value ? c : null", "  return c"),
    ("M21", "component: stale messages failure sets the current error", V,
     """  } catch {
    if (!isCurrent()) return
    convError.value = true""", """  } catch {
    convError.value = true"""),
    ("M22", "component: deselect does not advance the selection generation", V,
     """function deselectConversation() {
  selectionGen++
""", """function deselectConversation() {
"""),
    ("M23", "component: ending a lifetime does not advance the lifetime generation", V,
     """function endLifetime() {
  lifetimeGen++
""", """function endLifetime() {
"""),
    ("M24", "component: image fetch result accepted after its epoch ended (first guard)", V,
     "    if (epoch !== imageEpoch) return // ended while pending", "    // ended while pending"),
]

results = []
for mid, label, rel, old, new in M:
    out = os.path.join(WORK, f"{mid}.json")
    spec = json.dumps({"path": rel, "old": old, "new": new, "expected": 1})
    cmd = [sys.executable, "-B", RUNNER, "--project", PROJECT, "--work", WORK, "--label", mid.lower(), "--source", "live", "--out", out,
           "--mutate", spec, "--files"] + FILES
    subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    data = json.load(open(out, encoding="utf-8"))
    if data.get("disposition") == "NOT_APPLIED":
        results.append({"id": mid, "mutation": label, "disposition": "NOT_APPLIED", "detail": data["mutations"]})
        print(mid, "NOT_APPLIED")
        continue
    failed = [t for t in data["tests"] if t["status"] == "failed"]
    if failed:
        disposition = "KILLED"
    elif data["exitCode"] == 0 and data["counts"].get("passed", 0) > 0:
        disposition = "SURVIVED"
    else:
        disposition = "INCONCLUSIVE"
    results.append({
        "id": mid, "mutation": label, "path": rel, "disposition": disposition, "matchCount": data["mutations"][0]["actualMatches"],
        "beforeLfSha256": data["mutations"][0].get("beforeLfSha256"), "afterLfSha256": data["mutations"][0].get("afterLfSha256"),
        "exitCode": data["exitCode"], "counts": data["counts"], "suiteLevelFailures": data["suiteLevelFailures"],
        "failingTests": [{"file": t["file"], "title": t["title"], "assertion": (t["failure"] or "").split("\n")[0][:200]} for t in failed][:12],
        "failingTestCount": len(failed),
    })
    print(mid, disposition, len(failed))
json.dump({"schemaVersion": "r088.mutationRun@1", "liveWorktreeMutated": False, "results": results}, open(OUT, "w", encoding="utf-8"), indent=2, ensure_ascii=False)
