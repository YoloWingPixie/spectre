# Working with Spectre

Spectre is operated by an LLM agent to present specifications and audits to
its user. The agent prepares the content and starts the viewer. The user
reviews it in the browser; the agent reads saved feedback before continuing.
Keep that workflow explicit in documentation and user-facing behavior.

## Operating the tool

Read [SKILL.md](SKILL.md) when using Spectre to prepare a review for a user.
It covers spec mode and routes audit work to the bundled audit instructions.
Run the viewer in a persistent terminal, verify its printed URL, and present
that URL to the user. Read saved feedback when the user asks to continue.

## Repository boundaries

- `cmd/spectre` owns CLI mode selection and process startup.
- `internal/spec` owns Markdown parsing, spec pages, and append-only review files.
- `internal/audit` owns report validation, audit state, feedback, rendering,
  the local viewer, and audit-agent setup.
- `internal/webui` owns the shared header, visual styles, and theme preference.
- `docs/audit` holds the audit instructions, schemas, and starter report.
  Its Go file embeds these directly from their canonical location.
- `scripts/check-browser.mjs` exercises both modes in Chromium.

Follow the contracts and documented exceptions in
[docs/integration.md](docs/integration.md). Keep product commands in one Go
executable with no Node runtime or production dependencies. Browser JavaScript
is embedded; there is no frontend package-install or build step.

Preserve `.audit-report.json`, `AUDIT_REPORT_HOME`, stable audit/finding/option
IDs, saved feedback formats, and legacy `specview` prefix declarations. Keep
shared presentation in `internal/webui` when changing either mode's appearance.
The agent performs analysis and source edits; Spectre renders content and
stores feedback. Do not imply that starting a viewer runs an LLM or that
selecting an option automatically changes project code.

## Development commands

Use the existing Taskfile:

- `task run` starts the bundled spec example; set `DIR` for another spec.
- `task run:audit` starts the bundled audit example; set `PROJECT` for an
  initialized project audit. Both accept `PORT` and default to an available port.
- `task format` formats Go source.
- `task test:audit` checks audit and CLI behavior with race detection.
- `task verify` checks formatting, static analysis, race tests, audit fixtures,
  the build, and reachable vulnerabilities.
- `task test:browser BROWSER=/path/to/linux/chromium` checks browser behavior.
  Its test driver requires Node 22; installed Spectre does not.
- `task benchmark:audit` measures report linting.
- `task skill:check VALIDATOR=/path/to/skill-creator/scripts/quick_validate.py`
  validates the root and bundled audit skills.

Run build-producing tasks sequentially: several write the same `bin/spectre`.
The shared build runs once within each Task invocation. Demo feedback lives
in `bin/demo`; neither run task overwrites existing demo data.
