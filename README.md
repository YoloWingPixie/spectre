# Spectre

Spectre reviews Markdown specifications and JSON audit reports in a local
browser. Its default spec mode serves a Markdown folder for line-by-line
review. Every spec page is scoped to one chapter or section, so the
browser never holds the whole spec. You can search, follow ID links and
backlinks, answer open decisions, add comments and give a verdict on each
item. Your feedback goes into Markdown files next to the spec. Spectre
never edits the spec itself.

Both modes share Spectre's header, typography, colors, controls, and light
and dark themes. Theme choices persist across local viewers on the same host.

## Install and run

Build from a checkout with Go 1.26.6 or newer and [Task](https://taskfile.dev):

```sh
task build
./bin/spectre -dir path/to/spec -port 8765
```

The executable needs no Node runtime. Spec review is the default mode.
`spectre spec` accepts the same flags. Copy `bin/spectre` to a directory on
PATH to use `spectre` directly.

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `.` | spec folder; review files go to `<dir>/review/` |
| `-port` | `8765` | loopback port; `0` chooses an available port |

Spectre re-parses the Markdown whenever a spec file changes (it checks on
each request), so your edits show on the next reload.

`task verify` checks formatting, runs `go vet`, race tests, and `govulncheck`,
validates the audit examples, and builds `bin/spectre`. The Makefile delegates
to these tasks.
`task benchmark:audit` measures linting time and allocations for reports with
1 and 100 findings.
`task check:browser BROWSER=/path/to/chromium` checks audit feedback and
desktop and phone layouts. That optional development driver uses Node 22;
the Spectre executable does not.

## Browser examples

Prepare the bundled ATC specification and Flightline audit:

```sh
task demo:setup
```

In separate terminals, start each viewer:

```sh
task demo:spec
task demo:audit
```

Open [the spec example](http://127.0.0.1:8765/) and
[the audit example](http://127.0.0.1:8766/). Feedback stays under `bin/demo`.
Setup preserves existing demo feedback. Each viewer accepts `PORT=<number>`
to use another port. Stop a viewer with Ctrl+C.

## Optional audit mode

Audit mode uses the report format and browser interface from audit-report.
Both modes run in Go. Git is used to record repository changes; resume
reports `repositoryChanged: null` if Git state is unavailable.

```sh
spectre audit init --project path/to/project
spectre audit resume --project path/to/project
spectre audit view --project path/to/project
```

Initialization creates `.audit-report.json` in the project and saves reports,
feedback, and screenshots under the sibling `<project>.audits/<audit-id>/`
folder. Repeating `init` resumes the active audit. Use `--new` for a separate
audit, or `--source report.json` to import an existing report and its screenshots.
Existing audit-report state and feedback use the same files and IDs.

The viewer prints a loopback URL and saves decisions and notes to the audit
folder. Wait for **Saved to the audit folder** before closing the page.
Stop the server with Ctrl+C. Reports are re-read on each request.

```sh
spectre audit build report.json --strict
spectre audit build report.json --check --strict
spectre audit build --init report.json
spectre audit checkpoint --project path/to/project --checkpoint checkpoint.json --revision '<projectRevision>'
spectre audit import --project path/to/project --feedback feedback.json
spectre audit list
```

`build` writes a standalone HTML report. Small screenshots are embedded;
when screenshots exceed 3 MiB in total, send the matching `.assets` folder
with the HTML. Standalone reader feedback stays in the browser until exported
and imported. `--check` writes nothing. `--strict` rejects plain-English lint
warnings. A checkpoint contains `completed` and `nextSteps` arrays and an
optional `status` (`in-progress`, `awaiting-review`, or `complete`). Use the
`projectRevision` from the latest resume to avoid overwriting newer progress.

Set `AUDIT_REPORT_HOME` to change the discovery index from
`~/.audit-report/projects.json`. `--audit <id>` selects an audit for resume,
checkpoint, view, or import. Imports preserve saved records and reject conflicts;
use `--replace` only after reviewing both records. Saved decisions are feedback,
not permission for an agent to change application code.

For coding agents, register the included `spectre-audit` skill:

```sh
spectre audit setup --agent codex
```

Use `claude`, `gemini`, or `opencode` for another agent. Run `spectre audit setup`
without flags for a generic prompt. Registration preserves existing skills;
`--replace` applies only to links. `--remove` removes only this installation's
link. Keep Spectre on PATH. Setup and audit initialization extract embedded
schemas and instructions into a versioned user cache. Keep that cache for
schema completion and registered instructions.

Audit reports retain their POSIX path contract. Use WSL on Windows.
See [integration requirements](docs/integration.md),
[resume and recovery](internal/audit/docs/resuming-audits.md), and
[report schema](internal/audit/schema/report.schema.json).

## Spec folder

The folder must contain these six files:

| File | Contents |
|---|---|
| `README.md` | `# <Title>`, then free Markdown (shown at `/readme`) |
| `requirements.md` | `# <Title>`, then one `## <n>. <Chapter name> (<AREA>)` per chapter, with one `### <REQ-ID> · <title>` per requirement. A `##` section that has no requirements counts as part of the overview. |
| `design.md` | free sections (`##`, `###`). Each design element is a table row `\| DE-XXX \| Element \| Package / files \| Responsibility \| ...` |
| `scenarios.md` | `### S-nn · <title>` per scenario |
| `tasks.md` | `## Mn · ...` milestone sections with `#### T-Mn-nn · <title>` per task, plus a milestone table whose rows start `\| Mn ...` and whose last column lists exit scenarios (`S-01…S-21`, `S-26, S-27`) |
| `open-decisions.md` | `### OD-nnn · <title>` per decision. Add `(resolved)` to the title, or a `Resolution` field, to mark one resolved. |

Items carry fields as list lines `- **Name:** value`. Indented
continuation lines belong to the field. The viewer reads these fields:

- Requirements: `Priority` (`MVP`, `P2` or `P3`, optionally followed by
  `· **Status:** decided|derived|open ...`). `Design` lists the DE IDs.
- Tasks and scenarios: `Covers` lists the requirement IDs.
- Open decisions: `Options` (`(a) ...; (b) ...`, inline or as a list),
  `Recommendation`, `Blocks` (the requirement IDs that wait on it) and
  `Resolution`.

Milestones come from the tasks. A requirement is in each milestone whose
tasks cover it, and MVP requirements are also in M1. A scenario is in the
milestone whose exit criteria list it. An open decision is in the
milestones of the requirements it blocks.

## IDs

| Kind | Form | Example |
|---|---|---|
| Requirement | `<PREFIX>-(FR\|NFR)-<AREA>-nnn` | `ATC-FR-IDN-001`, `ASOC-NFR-PERF-002` |
| Scenario | `S-nn` (sub-step `S-nna`) | `S-01`, `S-06b` |
| Task | `T-Mn-nn` | `T-M1-04` |
| Open decision | `OD-nnn` | `OD-017` |
| Design element | `DE-<NAME>` | `DE-FAC` |

`PREFIX` is 2 to 8 characters (`[A-Z][A-Z0-9]{1,7}`). The spec's prefix is
the one its README declares:

```markdown
<!-- spectre: prefix=XYZ -->
```

Legacy `<!-- specview: prefix=XYZ -->` declarations remain supported.

If the README declares no prefix, Spectre uses the most common prefix
among the requirement IDs. If there are no requirements, it uses `ATC`.

Every ID in the text links to `/id/<ID>`. A short ID such as `IDN-002` or
`PERF-002` takes the prefix and FR/NFR of the last full ID before it, or
the spec's prefix if no full ID comes first. A short ID that matches only
one known requirement links to that requirement.

## Titles

Page titles use the spec's short name. That is the README title before
its first `:`, without a trailing ` role`: "ATC role: specification
(SDD)" gives "ATC", and "Thing: spec" gives "Thing". If there is no colon,
the whole title is the name. If there is no title, the prefix is the name.
The header shows "<Name> Spec", and the index title is
"<Name> Spec Review".

## Pages

| Path | Shows |
|---|---|
| `/` | counts, review progress, chapter list |
| `/req`, `/req/<n>` | the requirements overview; one chapter per page, with a feedback form for each requirement |
| `/id/<ID>` | one item with its backlinks, comments, feedback history and forms |
| `/design/<n>`, `/scenarios/<n>`, `/tasks/<n>` | the documents, split into pages by section |
| `/od` | open decisions, filtered by open / unanswered / answered / resolved |
| `/search` | word search with filters for kind, priority, status, milestone and review state; split into pages |
| `/next?kind=req&after=<ID>` | the next unreviewed item (add `&in=page` to open it in its chapter) |
| `/readme` | the README |

## Review output

Spec mode only appends, and only to `<dir>/review/`. Each spec folder gets
its own set of files:

| File | Written by | Entry |
|---|---|---|
| `feedback.md` | the Accept / Change / Reject / Question buttons | `## <ID> · <UTC time>`, `Verdict`, `Text` |
| `comments.md` | the comment form on `/id/<ID>` | `## <ID> · <UTC time>`, `Comment` |
| `answers.md` | the answer form on `/id/OD-nnn` | `## <ID> · <UTC time>`, `Choice`, `Answer`, `Note` |

A new file starts with `# <Name> spec review: feedback` (or `comments` or
`answers`). Multi-line text is written as a block quote. The page shows the
latest verdict for each ID and keeps the earlier ones as history. An open
decision with an entry in `answers.md` shows as "answered (pending
fold-in)" until you copy the answer into `open-decisions.md` by hand. Then
delete or archive the entry.

## Security

The server binds to 127.0.0.1 and rejects requests with a foreign `Host`
header and cross-origin POSTs.

## License

AGPL-3.0; see [LICENSE](LICENSE).
