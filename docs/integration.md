# Spectre integration requirements

Spectre combines Markdown spec review with the audit-report workflow in one Go executable. Spec review remains the default mode. Audit mode is selected with `spectre audit`.

The intended operator is an LLM agent working for a user. The agent prepares
the specification or audit report, starts Spectre, and presents the review
URL. The user records feedback in the browser. The agent reads that feedback
when asked to continue and updates the work within the user's instructions.
Spectre owns presentation and feedback storage; the agent owns analysis and
requested source changes. See [the operating skill](../SKILL.md) and
[repository instructions](../AGENTS.md).

## Definitions and constraints

A spec is the existing six-file Markdown folder. An audit contains a JSON report, reader feedback, and a checkpoint. The existing `.audit-report.json` dotfile owns audit identities and points to the sibling audit directory.

- CON-001: Both modes shall run without Node or the audit-report checkout.
- CON-002: The integration shall preserve user files and the source audit-report repository.
- CON-003: The integration shall add no production dependency.
- CON-004: The module shall require Go 1.26.6 or newer.

## Functional requirements

- REQ-001: Spectre shall serve the existing spec review workflow by default.
- REQ-002: Audit mode shall preserve existing report, checkpoint, and feedback formats.
- REQ-003: Audit mode shall provide initialization, resume, checkpoint, list, feedback import, local viewing, and standalone HTML generation.
- REQ-004: The project command, module, build output, and documentation shall use the Spectre name.
- REQ-005: The spec parser shall accept both `spectre` and legacy `specview` prefix declarations.
- REQ-006: Audit mode shall provide agent instructions through an optional setup command.
- REQ-007: Spec and audit views shall use one shared application header, typography, palette, and control style.
- REQ-008: Theme selection shall persist across spec and audit views on the same local host.
- REQ-009: Feedback export shall include current unsaved edits when browser storage fails.
- REQ-010: Save deduplication shall compare complete normalized feedback content.
- REQ-011: A finding without courses of action shall retain its feedback controls and saved feedback.
- REQ-012: Report validation shall reject unknown keys that case-fold to declared field names.
- REQ-013: Numeric validation shall preserve representable integer values exactly and reject fractional integer fields.
- REQ-014: File-reference line numbers shall use decimal notation, including leading-zero input.
- REQ-015: Initialization shall return the audit identity selected by its state update.
- REQ-016: Copied project identities shall be rejected before modifying that project's audit or state files.
- REQ-017: Checkpoint shall reject unreadable feedback before committing project state.
- REQ-018: First initialization shall validate the project index before creating audit files.
- REQ-019: Every case variant of a feedback record identity shall match its finding key.
- REQ-020: Option-note lookup shall use only explicitly stored properties, including for identifiers such as `constructor`.
- REQ-021: Local restoration shall merge saved feedback with pending edits without overwriting edited controls.
- REQ-022: Write responses shall be constructed from the operation's captured audit data without repeating registration.
- REQ-023: Scenario headings before the first section shall retain item identity, search, and review behavior.
- REQ-024: Search highlighting shall emit each original snippet segment once, including for repeated or overlapping terms.
- REQ-025: A checkpoint canceled during repository discovery shall leave saved project state unchanged.
- REQ-026: Standalone feedback shall be isolated by stable report identity rather than title and commit.
- REQ-027: Legacy browser feedback shall require reader confirmation before import into an identified report.
- REQ-028: Unexpected audit viewer failures shall record their cause locally without exposing feedback or tokens.

## Quality requirements

- NFR-001: A failed or conflicting save shall preserve the last saved audit data.
- NFR-002: Audit HTTP access shall be restricted to the viewer's loopback origin.
- NFR-003: Viewer feedback writes shall require a per-viewer token and current report and feedback revisions.
- NFR-004: A viewer shall release its audit lock after normal shutdown.
- NFR-005: Report rendering shall escape user text and preserve browser filters, editor links, decisions, and notes.

## Design

The command selects `internal/spec` or `internal/audit`. Each package owns its own input format and feedback storage. Audit schemas and browser assets come from audit-report. Go owns report validation, rendering, storage, HTTP serving, and agent setup. JavaScript runs only in the browser.

Audit instructions, schemas, and the starter report live in `docs/audit`.
Its small Go package embeds those files directly. This source-layout exception
to GO-011 keeps one authoritative copy under `docs`; `go:embed` cannot read
files from a parent directory. Runtime implementation remains in `internal`.

Retain the existing CSS and browser scripts. The artifact contract and the no-dependency constraint take precedence over UI-012 and UI-013 and their styling and application-layer rules. This exception covers the embedded spec and audit pages. The integration does not introduce a frontend build pipeline.

Shared presentation belongs to `internal/webui`. It owns the application header, visual tokens, base controls, and theme preference. The mode packages retain their document layouts and feedback behavior. Both modes use the spec viewer's system fonts, neutral surfaces, blue accents, and rounded controls. This shared boundary centralizes the application style under ARCH-002/014 and UI-001/011; the existing CSS exception remains in effect.

Keep `.audit-report.json`, `AUDIT_REPORT_HOME`, and `~/.audit-report/projects.json` for compatibility. New skill registration uses `spectre-audit` and does not replace an existing audit-report registration. Embedded schemas and agent instructions are materialized in a versioned user cache when needed.

Agents run the installed Spectre executable directly. Task is the development interface. This is a scoped TASK-007 exception for distribution as one Go executable; development and verification tasks call the same Go command.

These boundaries follow ARCH-001/002/020/027, IMPL-005/013, DM-004/005, and GO-010/011/012. Repository checks follow TASK-001/004 and TEST-005/012.

## Acceptance and verification

| Criterion | Requirements | Method |
|---|---|---|
| AC-001: Existing spec tests pass and both prefix declarations load. | REQ-001, REQ-005 | Test |
| AC-002: Imported audit-report fixtures build and saved audits resume with their identities and feedback. | REQ-002, REQ-003, NFR-005 | Integration test |
| AC-003: Stale revisions, invalid input, foreign origins, and failed writes do not replace saved data. | NFR-001, NFR-002, NFR-003 | Test |
| AC-004: Viewer shutdown removes its lock and permits restart. | NFR-004 | Integration test |
| AC-005: The built Spectre command runs both modes without Node or the sibling checkout. | CON-001, REQ-004 | Test and artifact inspection |
| AC-006: Setup preserves existing registrations and provides usable commands and schemas. | REQ-006, CON-002 | Test |
| AC-007: No production dependencies or source-project edits appear in the complete diff. | CON-002, CON-003 | Inspection |
| AC-008: The supported Go toolchain passes the vulnerability scan. | CON-004, GO-008 | Test and artifact inspection |
| AC-009: Spec and audit pages have equal computed base styles, share theme selection, and fit 1280px and 400px viewports. | REQ-007, REQ-008, NFR-005 | Browser test and visual inspection |
| AC-010: Blocked standalone storage, colliding old hashes, and removed options preserve feedback through saves and export. | REQ-009, REQ-010, REQ-011 | Browser regression test |
| AC-011: Alias keys and fractional numbers fail validation; exact large integers, integral decimal/exponent forms, and decimal references retain their values. | REQ-012, REQ-013, REQ-014 | Test |
| AC-012: Interleaving initialization and response collection retains the selected identity; copied-project init and checkpoint conflicts leave state, index, and audit files unchanged. | REQ-015, REQ-016, NFR-001 | Test |
| AC-013: Corrupt feedback and invalid initial indexes fail before state creation or mutation; response collection does not repeat registration; conflicting identity aliases preserve saved bytes. | REQ-017, REQ-018, REQ-019, REQ-022 | Test |
| AC-014: Inherited property names remain blank until explicitly edited; delayed local loading retains drafts, untouched controls, and retired feedback through subsequent saves and export. | REQ-020, REQ-021 | Browser regression test |
| AC-015: Introduction scenarios remain addressable; repeated, overlapping, and escaped search terms produce bounded valid highlighting. | REQ-023, REQ-024 | Test |
| AC-016: Canceled checkpoints preserve state bytes; unauthorized PUTs and rejected copied-project operations preserve feedback and report bytes. | REQ-025, REQ-016, NFR-003 | Test |
| AC-017: Same-title standalone reports keep separate feedback; legacy import requires confirmation and preserves pending edits. | REQ-026, REQ-027 | Browser test |
| AC-018: A viewer filesystem failure records its cause locally and returns a generic browser error. | REQ-028 | Integration test |

Run `task test:audit` for the changed workflow and `task verify` for formatting, static analysis, race tests, vulnerability analysis, and build. Run `task test:browser` with Chromium for disk feedback, restart, conflicting tabs, and layouts at 1280px and 400px.

Run `task benchmark:audit` to measure linting time and allocations with 1 and 100 findings. Fixed phrase rules compile once, while matching and diagnostic order remain covered by regression tests. Multiline sentence tests cover LF, CRLF, and blank-line separators. Feedback tests cover case-insensitive null-note rejection and preservation of saved data when an import cannot read its report (NFR-001, AC-003).

The minimum Go patch addresses standard-library findings from `govulncheck`, including the `html/template` and `net/http` paths used by Spectre (GO-008, CON-004).

## Limits and risks

This is a local single-user workflow. It does not synchronize audits across devices. A saved reader decision does not authorize a code change.

Project state and the project index use separate atomic file replacements. Initialization holds the index lock across state creation and index update. A final index-file I/O failure can still occur after state persistence; the two files do not form a crash-atomic transaction.

The canonical checkout is `/mnt/c/git/spectre`. Its `origin` is the private repository `https://github.com/YoloWingPixie/spectre.git`. The repository was created empty; no project files were committed or uploaded. Repository-local Git credentials use the authenticated GitHub CLI.

Windows prevented renaming `C:\git\specview` because another process held it open. A copy at `C:\git\spectre` was verified against all original files and Git metadata. The locked original remains at `C:\git\specview`. Open the new checkout for further work to keep one active working tree.

Legacy reports use POSIX paths. Windows audit workflows require WSL. Existing source files and audit data are not migrated automatically. Git supplies repository fingerprints; resume reports an unknown fingerprint when Git state is unavailable. Hosted artifact feedback requires its external runtime and cannot be proved by local tests.

## Completion evidence

AC-015 through AC-018 pass with the audit fixes. `task verify` covers scenario
identity, bounded Unicode-safe highlighting, canceled checkpoints, rejected
writes, persistent report IDs, and local failure diagnostics. Linux Chromium
checks cover report isolation, confirmed legacy import, storage failure, and
draft preservation. Windows Chrome now fails before fixture setup with the
Linux Chromium prerequisite message. These checks apply IMPL-008, TEST-006/025,
and GO-027/030/036; hosted artifact behavior remains unverified.

AC-001 through AC-008 passed through `task verify`, `task test:audit`, `task test:fuzz`, browser checks, `task skill:check`, and inspection. The built executable reports Go 1.26.6 and `CGO_ENABLED=0`. The binary integration test runs default spec mode, explicit spec mode, audit initialization, resume, standalone generation, and both viewers with an empty PATH.

The Chromium check exercised disk saving, restored notes and decisions, conflicting tabs, locked writes, retry, offline edits, and server restart. Viewer and standalone layouts passed at 1280px and 400px. The optional development browser driver uses Node 22; product commands do not.

AC-009 passed with matching computed body, heading, header, and theme-control styles in spec and audit views. A theme change in spec mode persisted into audit mode on another port. Screenshots of both modes were inspected at 1280px and 400px. Both live demo servers were restarted with the shared presentation.

AC-010 through AC-012 passed through the expanded browser suite and Go regression tests. The browser checks exercise storage-failure export, distinct content with colliding legacy hashes, and retained notes and decisions after all options are removed. Go tests cover exact numeric parsing, alias rejection, decimal references, copied-project conflicts, and initialization interleaving at the persistence/response boundary.

AC-013 and AC-014 passed after reproducing their failures. Go regressions cover corrupt feedback, malformed initial indexes, identity aliases, and an index lock introduced during response collection. Browser regressions cover inherited option identifiers and edits made while local feedback initialization is delayed, including retained metadata and later saves.

The source audit-report working tree remains clean. The license and spec fixtures match the original snapshot. Hosted artifact capabilities and native Windows runtime behavior were not exercised.

## Changed files

- Root: `.gitignore`, `AGENTS.md`, `SKILL.md`, `Makefile`, `README.md`, `Taskfile.yml`, `go.mod`, `docs/integration.md`.
- Command: `cmd/specview/main.go` moved to `cmd/spectre/main.go`; `cmd/spectre/main_test.go` added.
- Spec package: `internal/specview/` moved to `internal/spec/`. Its Go files are `asoc_spec_test.go`, `link.go`, `link_test.go`, `markdown.go`, `review.go`, `review_test.go`, `search.go`, `server.go`, `server_test.go`, `spec.go`, and `spec_test.go`. Branding and theme changes also affect `assets/app.js` and `assets/tmpl/layout.html`. Other templates, CSS, and spec fixtures moved without content changes.
- Audit package: `internal/audit/model.go`, `schema.go`, `report.go`, `render.go`, `storage.go`, `feedback.go`, `viewer.go`, `build.go`, `command.go`, `setup.go`, `resources.go`, `lint.go`, `audit_test.go`, `schema_test.go`, `render_test.go`, `lint_test.go`, `correctness_test.go`, `persistence_test.go`, and `feedback_fuzz_test.go` added. The schema engine owns validation shared by report and project-state parsing; report-specific checks remain in `report.go`.
- Audit assets: `internal/audit/assets/style.css`, `script.js`, `feedback.js`, `links-helpers.js`, `feedback-helpers.js`, and `report.html` added.
- Audit schemas and starter: `docs/audit/schema/report.schema.json`, `docs/audit/schema/project-state.schema.json`, and `docs/audit/templates/report.starter.json`.
- Audit example: `internal/audit/testdata/example/report.json` and `internal/audit/testdata/example/shots/wiki-desktop-night.jpg`.
- Audit instructions: `docs/audit/SKILL.md`, `docs/audit/checklist.md`, `docs/audit/resuming-audits.md`, `docs/audit/writing-guide.md`, and `docs/audit/commands.md`. `docs/audit/embed.go` bundles the runtime instructions and schemas, including the audit checklist.
- Browser verification: `scripts/check-browser.mjs` added.
- Shared presentation: `internal/webui/webui.go`, `assets/header.html`, `assets/style.css`, and `assets/theme.js` added. Integration changes affect `internal/spec/server.go`, `assets/tmpl/layout.html`, `assets/style.css`, and `assets/app.js`; `internal/audit/render.go`, `assets/report.html`, and `assets/style.css`; `scripts/check-browser.mjs`, `Taskfile.yml`, `README.md`, and this document.
