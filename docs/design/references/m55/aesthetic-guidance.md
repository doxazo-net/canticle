# #1339 next/ Settings -- aesthetic / layout guidance (user, 2026-06-02)

STATUS: PARTIAL -- user has MORE aesthetic/layout guidance to give before #1339
is built; this captures what was shared so far. Revisit/finish this conversation
(brainstorming) BEFORE starting the #1339 N1/N2 chrome. The frozen spec
`06-settings.md` + the prototype remain the baseline; this guidance SUPPLEMENTS
the prototype where the prototype "doesn't fully capture" the intended look
(charter: spec wins for functional/structural; user guidance + prototype drive
chrome aesthetics).

## Inspiration: Firefox `about:preferences`

User's stated inspiration for the searchable Settings screen. "I really like how
FF is laid out." Chrome's searchable settings noted too, but FF is the model.
Cross-referenceable from knowledge; published Mozilla screenshots can be fetched
for exact fidelity (Playwright drives Chromium, so it can NOT open live FF
`about:` pages).

### FF traits to emulate
1. Search input pinned ABOVE the category rail (top-left), magnifier glyph --
   first thing the eye lands on, not buried in the content pane.
2. Search filters across ALL categories at once: highlights matches in place
   (soft highlight bar) AND surfaces a synthetic "Search Results" entry; flags
   the rail category that contains a hit. (Richer than plain show/hide.)
3. ONE long scrollable content column; clicking a rail category JUMP-SCROLLS to
   that section's anchor -- not separate pages, not accordion panels that hide
   the rest.
4. Calm row rhythm: label + muted secondary description + control, grouped under
   bold subsection headings with thin dividers and generous whitespace.

## Decision gaps vs the current plan (resolve before/with the user)

The D-migration plan already specs a 260px sticky rail (4 groups: Essentials /
Data / Integrations / System) + keyword filter over labels+help with
`localStorage` persistence -- bones match FF. Open deltas the prototype does not
settle:

- **Rail behavior:** FF jump-scroll-to-anchor within ONE page vs the prototype's
  collapsible groups that show one group at a time. (User leans FF.)
- **Search behavior:** FF highlight-in-place + a "Results" view vs a plain
  show/hide keyword filter.
- **Search placement:** above the rail (FF) vs wherever the prototype parks it.

## Converging decisions (dialogue 2026-06-02 -- not yet a locked spec)

Direction both the user and the agent lean toward for the `next/` chrome:

- **Single long scrollable settings page**, FF-style, with the sticky rail acting
  as **anchors + scroll-spy** (jump-scroll to `#section-id`, rail highlights the
  current section) -- NOT discrete panel swaps. It is a document you scroll, not
  an app with modes.
- **`/`-triggered in-app search box is the PRIMARY search** (the global `/`
  keyboard-registry trigger). It reads the STATIC index
  (`window.swSettingsSearchIndex` / `BuildSettingsSearchIndex`), so it finds
  matches in every section regardless of render state. This deliberately makes
  native `Ctrl+F` a non-goal -- which is what unblocks lazy-loading below.
- **Lazy-load the heavy CRUD bodies** (provider keys, connections, libraries,
  webhooks, API tokens, users, rule config forms) via declarative
  `hx-trigger="revealed"` (or on expand), so the long page stays lean. Render
  every section SHELL (header + light World-2 KV controls) eagerly for instant
  anchor/scroll/search; hydrate the expensive bodies on reveal. `hx-trigger=
  "revealed"` is declarative HTMX, not hand-written JS, so it should clear the
  minimal-JS bar -- confirm at build time.
- Two seams that must hold for the above to work (add tests):
  1. The static search index must stay COMPREHENSIVE -- every section's controls
     represented. It is a hand-list today, so "search finds everything" is a
     maintenance contract. A test should fail if a section has zero index entries.
  2. Selecting a `/`-search result must REVEAL + scroll the (possibly lazy)
     target section, not scroll to an empty shell.

### DECIDED (user, 2026-06-02)

- **Layout: the long single-page model with the left rail tracking the user's
  position in the UI (scroll-spy).** This is the chosen `next/` Settings
  direction -- not discrete panel swaps.
- **A/B is PoC-only, not long-term.** The user wants the stable-vs-next
  comparison only as a proof-of-concept before this new version ships, NOT an
  ongoing A/B feature. So: do NOT build two next/ variants; build the long-page
  variant as next/ and compare it to stable for the pre-ship PoC via the existing
  `SW_UX` flag. No sub-flag, no second next/ shell.

STATUS: user signalled they may have MORE aesthetic/layout guidance later;
the core layout decision (long-page + scroll-spy rail + `/`-search + lazy CRUD
bodies) is now settled. Promote to a 06-settings.md addendum when #1339 N1/N2
work begins.

## Bearing on #1809 (confirmed: none)

The anchor-vs-panel + lazy-load decisions are ALL `next/` chrome (#1339)
concerns. #1809 extracts each `.sw-card` into a chrome-agnostic `Section*(data)`
func that renders identical markup however a chrome arranges it; tab-`hidden`
toggling and panel wrapping stay in the STABLE chrome wrapper, not in the section
funcs. So S1-S4 proceed byte-identical-to-stable and are unaffected. One thing to
NOT do during extraction: do not bake tab-panel-only assumptions or conflicting
DOM ids into the shared funcs (the `next/` chrome supplies its own section-anchor
wrapper + `hx-trigger="revealed"` hooks).
