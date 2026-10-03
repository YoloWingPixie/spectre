# Audit commands

For the agent workflow, checkpoints, and recovery, read the
[resume guide](resuming-audits.md).
Use the [audit checklist](checklist.md) to select checks for the audit scope.

```sh
spectre audit init --project path/to/project
spectre audit resume --project path/to/project
spectre audit view --project path/to/project
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

Audit reports retain their POSIX path contract. Use WSL on Windows.
See [integration requirements](../integration.md) and the
[report schema](schema/report.schema.json).

## Agent setup

`spectre audit setup --agent codex` registers the bundled `spectre-audit`
skill. Supported agents are `codex`, `claude`, `gemini`, and `opencode`.
Run setup without flags for a prompt pointing to the extracted instructions.
`--skills-dir` selects a custom skills directory. `--replace` replaces a
skill link after review; `--remove` removes only this installation's link.
The root [Spectre skill](../../SKILL.md) covers both modes and can be read directly
from the checkout; this setup command registers the audit-only skill.

Setup and audit initialization extract embedded schemas and instructions
into a versioned user cache. Keep that cache for schema completion and
registered instructions.
