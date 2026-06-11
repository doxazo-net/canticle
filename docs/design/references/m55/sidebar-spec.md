# #1778 + #1715 -- next/ Sidebar Redesign: LOCKED SPEC (2026-06-04)

Authoritative design for the `sidebar` teammate to bring to completion. The lead (UX) iterated this live with the maintainer; this doc + the existing spike are the source of truth. Anchors #1778 (semantic glyphs / chrome), folds in #1715 (Reports children nesting).

## Current state: a working SPIKE (commit it first)
- Worktree `../stillwater-m55-1778-sidebar-spike`, branch `m55-1778-sidebar-spike` (off main `6520e522`). **UNCOMMITTED.** It BUILDS and runs on :1973 (SW_UX=dual).
- FIRST STEP: commit the existing spike as a checkpoint (`feat(next/chrome): sidebar redesign spike (Part of #1778)`), then continue. Rename the branch to `m55-1778-sidebar` before the PR.
- Files already changed in the spike (read them for exact current code):
  - `web/templates/next/sidebar.templ` -- full rewrite; no longer delegates to stable. Reuses the stable `.sw-sidebar-*` classes for behavior.
  - `web/templates/next/sidebar_helpers.go` -- new `sidebarName` / `sidebarInitial`.
  - `web/static/css/input.css` -- `.sw-sidebar-subnav-link` indent, `.sw-sidebar-actions`, brand/version, light-theme tree-line override, resize easing + motion/lite snap gating.
  - `web/static/js/sidebar.js` -- `/next`-aware `highlightActiveLink` (strips `/next` so `data-path` matches on `/next/*`).

## Locked design (top -> bottom)
- **Brand header:** favicon mark + "Stillwater" on line 1; **version on its own muted line below** in JetBrains Mono (`var(--sw-font-mono)`, ~0.625rem, ellipsis) so nightly/dirty strings fit. **Collapse chevron top-right**, reuses `swSidebar.cycle()` (full -> icon-only -> hidden).
- **Primary nav** (semantic glyphs from `design/shell.jsx`; the **active-highlight is the screen indicator** since per-page H1s are dropped):
  - Dashboard (home) -> `/next/` + action/notif count pill
  - Artists (people) -> `/next/artists` + artist-count pill
- **REPORTS** = a **section header** (uppercase muted label, NOT a link -- this kills the old Reports/Compliance duplication). Children:
  - Compliance (bar-chart glyph) -> `/next/reports/compliance` -- ALWAYS shown
  - Duplicates (copy/overlap glyph) -> `/next/reports/duplicates` -- CONDITIONAL (only when count>0; hydrated via `/api/v1/reports/duplicates/count`), count shown as a count pill
  - Foreign Files (file-question glyph, SVG below) -> `/next/foreign` -- CONDITIONAL (only when count>0), count pill
- **Activity** (pulse) -> `/next/activity`
- spacer (`flex-1`)
- **Low-frequency destinations** (no label): Logs (doc) -> `/next/logs` (stub until #1338); Settings (gear, admin-only) -> `/next/settings` + update-dot; Preferences (sliders) -> `/next/preferences`
- **Bottom bar:** glyph-only quick **actions row ABOVE the user identity**:
  - actions: Theme (sun/moon, `swSidebar.cycleTheme()`), Help (`?`, `toggleHelpOverlay()`), Log out (hx-post logout). **Even spacing (`space-evenly`) at full width; centered vertical STACK in icon-only/narrow.**
  - user identity: avatar + name/role (name/role hidden in icon-only)

**All nav links -> `/next/<screen>`; keep `data-path` = the STABLE path** (e.g. `/artists`, `/reports/compliance`) so the `/next`-aware `highlightActiveLink` matches. No channel leak.

## Foreign Files glyph -- LOCKED (2026-06-05): plain dog-eared document
Maintainer chose the PLAIN dog-eared `document` glyph (Heroicons style) -- NO `?` overlay. Add as a Heroicons-style `Icon*` component (e.g. `IconDocument`). Keep Logs as `document-text` (dog-eared doc WITH text lines) so the two stay visually distinct; they sit in different sections (Foreign Files under REPORTS, Logs in low-freq) which further disambiguates. Discard the interim file-question SVG.

## Icon library (IMPORTANT -- the spike is off-standard here)
The project uses **Heroicons (v2)** vendored as inline-SVG `Icon*` templ components in `web/components/icon.templ` (e.g. `IconCrop`, `IconScissors`, `IconPhoto`, `IconMagnifyingGlass`). NO runtime icon dependency (minimal-JS rule).
- The spike currently inlines raw SVG paths (mix of `design/shell.jsx` hand-drawn glyphs + ad-hoc Heroicons). CONVERT the sidebar to use `Icon*` components, and ADD the missing semantic glyphs to `icon.templ` as new Heroicons-style components: home (Dashboard), user-group (Artists), chart-bar (Compliance), signal/activity (Activity), document-duplicate (Duplicates), document-question (Foreign Files -- dog-eared doc + `?` overlay; maintainer to supply an example), document-text (Logs), cog (Settings). Reuse existing `Icon*` where present.
- Foreign Files glyph: dog-eared `document` base + a `?` overlay in Heroicons stroke style. Interim SVG below; swap for the maintainer's example when relayed.

## Final tweaks to implement (this round)
1. **Duplicates count = consistent count pill.** Align the sidebar count pills (Artists, Dashboard, Duplicates, Foreign Files) to ONE canonical pill style. (The dashboard-vs-artists in-content pill inconsistency is a separate tracked follow-up; here just keep the sidebar's pills internally consistent.)
2. **Foreign Files conditional sub-item under Reports** (glyph above), shown only when count>0. Needs a foreign-files **count source** -- mirror `/api/v1/reports/duplicates/count`; if none exists, add a tiny count endpoint OR show unconditionally until #1773. Link `/next/foreign` (stub until #1773 ports the page; mirror the Logs stub approach).
3. **Bottom actions evenly spaced** (`space-evenly`) at full width; centered vertical stack in icon-only (already partly wired -- finish it).

## Hardening (bring-to-completion)
- **Typography pref:** bind sidebar fonts/sizes to the type tokens (`--sw-font-sans`/`--sw-font-mono` + the rem type-scale); mirror `.sw-next-artist-detail`. Replace hardcoded px. (Spike already uses `--sw-font-mono` for the version line.)
- **Background-opacity pref:** bind the sidebar surface to the bg_opacity pref; mirror `.sw-next-artist-detail`'s correct opacity impl (maintainer-cited reference). Verify BOTH themes at 85% AND ~50%.
- **i18n:** replace the spike's hardcoded English labels with i18n keys (some `nav.*` exist; add `nav.activity`, `nav.logs`, `nav.reports`, `nav.reports.foreign`, etc.). `en.json` is a hot-spot -- this PR owns its keys.
- **a11y (hard gate):** axe-core; contrast in BOTH themes; nav landmark + `aria-current` on the active item; proper labels/roles on the glyph-only buttons (theme/help/logout already have aria-label); keyboard focus order; verify icon-only too.
- **Count fragments:** the hydrated `/api/v1/reports/duplicates/count` fragment must emit the copy glyph + `sw-sidebar-subnav-link` + point `/next/reports/duplicates` (the spike renders Duplicates statically). Same pattern for the foreign-files count.
- **Motion/lite:** resize easing + snap under `prefers-reduced-motion` / `[data-motion=on]` / `[data-lite=on]` is done in the spike -- verify it holds after the actions-row + foreign-files changes.
- **icon-only edge cases:** version line, actions stack, count pills, section-label hide -- all collapse cleanly.
- **Tests:** templ render tests (structure, glyphs, conditional children present/absent), the `highlightActiveLink` `/next` fix, a11y.
- Keyboard g-leader nav is OUT OF SCOPE (owned by #1775); the sidebar only exposes nav targets. Logs is a stub (#1338).

## Constraints (teammate)
- NO push / PR / merge / gh mutations without an explicit HUMAN go. Commit locally freely.
- After `.templ` edits: `go tool templ generate` (never bare templ). Capture `go test -race` to a log. No emoji, no em-dashes.
- **Merge seams** (lead reconciles at merge): `input.css`/`styles.css`, `en.json`, and `layout.templ`/`handlers.go` (AssetPaths) overlap with the 4B + dashboard branches. Keep sidebar edits scoped to `.sw-sidebar-*`.
- Then `/prep-pr` + `pr-review-toolkit:review-pr`; report to the lead for the human PR go.
