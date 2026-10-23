# CVF Agent Declaration & G0 Probe Scratchpad

## Context
- Project: Customer-Care-Monitor-AI
- CVF Core: ../.Controlled-Vibe-Framework-CVF (commit e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba)
- Phase: BUILD / WORK_ORDER (CCMAI-RUNTIME-139)
- Risk Ceiling: R2
- Live Evidence Required: YES
- Active Role: IMPLEMENTATION_WORKER / BROWSER_AGENT
- Next Allowed Move: Evaluate tool declarations and execute G0 probe or stop per rule.

## Tool Inspection Results
- Inspected available browser tools: `open_browser_url`, `list_browser_pages`, `read_browser_page`, `capture_browser_screenshot`, `browser_get_dom`, `click_browser_pixel`, `browser_press_key`, `browser_move_mouse`, `browser_mouse_down`, `browser_mouse_up`, `browser_scroll`, `browser_drag_pixel_to_pixel`, `capture_browser_console_logs`, `browser_list_network_requests`, `browser_get_network_request`, `wait`, `browser_resize_window`.
- Checked for explicit primitives:
  - Create NEW owned tab with returned unique tab ID: `open_browser_url` takes `PageIdToReplace: "new_page"`, but does not return a unique tab ID natively as an explicit return value.
  - Close exact owned tab ID: Explicit own-tab closure primitive (`browser_close_tab` or similar) is completely **absent**.

## Decision per Rule 2
- Explicit new-tab creation with returned tab ID and explicit own-tab closure primitives are absent.
- Therefore, per Rule 2: STOP IMMEDIATELY. Do NOT navigate to any URL. Do NOT touch, inspect, click, or close any existing browser tabs.
- Final report will list available tools and state that explicit tab ID creation/closure primitives are absent.
