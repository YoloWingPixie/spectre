# Resume an audit

This guide is for the agent operating Spectre on the user's project. You
prepare the audit, present it in the browser, and read the user's saved
feedback to continue. The user reviews findings and chooses responses;
Spectre stores that feedback for you to retrieve.

The installed `spectre-audit` skill uses `spectre audit` from PATH. Keep the audited project as your working directory. Quote paths containing spaces. The Go executable contains the engine, schemas, and browser assets. Node, npm, and Task are not required at runtime.

## Start or resume

```sh
spectre audit init --project /path/to/my-project
spectre audit resume --project /path/to/my-project
```

Initialization creates `.audit-report.json` in the project and an audit directory in `../my-project.audits/`. Running it again resumes the active audit. Add `--new` only when starting a separate audit. Add `--source /path/to/report.json` to import an existing report into a new audit; its screenshots are copied with it. The source files are not changed.

The initial report has no findings. Set its scope and areas, then add verified findings. Give findings and options stable IDs. Missing option IDs prevent the persistent viewer from opening; position-based IDs could attach a decision to a different option after reordering.

Resume prints JSON with:

- `projectRevision`, needed when saving a checkpoint.
- `audit.checkpoint`, with completed work and next actions.
- `paths.report` and `paths.feedback`, the authoritative files.
- `report`, `reportErrors`, and `reportWarnings`.
- `feedback` and `feedbackReview`, including feedback for removed findings or options.
- `repositoryChanged`, comparing the repository with the checkpoint. It includes tracked and untracked file content. `null` means Git state was unavailable; it does not mean unchanged.
- `reportChanged`, indicating edits since the checkpoint.
- `pathChanged`, indicating that `report.repoRoot` needs updating after a move. The local viewer uses the current project path for editor links.

Read the saved context before investigating again. If code changed, verify the affected evidence. Feedback records decisions; it does not grant permission to implement them.

## Save progress

Write a checkpoint input file:

```json
{
  "completed": ["Verified token expiry behavior"],
  "nextSteps": ["Check whether the same behavior affects API tokens"],
  "status": "in-progress"
}
```

```sh
spectre audit checkpoint --project /path/to/my-project --checkpoint /path/to/checkpoint.json --revision "<projectRevision>"
```

Use the revision returned by the latest resume or checkpoint. A stale revision fails instead of replacing newer progress. The command records the current repository fingerprint and report revision. Save report edits before saving the checkpoint. Checkpoints replace the completed-work and next-action lists, so retain relevant earlier entries.

Supported states are `in-progress`, `awaiting-review`, and `complete`. Save after meaningful investigation, after incorporating feedback, and before handing control back. Keep checkpoints factual; they are work records, not conversation transcripts.

## Collect decisions

```sh
spectre audit view --project /path/to/my-project
```

Run the viewer in a persistent terminal, verify the printed local URL, and
give it to the user with a short explanation of what needs review. Keep the
command running while they use the page. If your environment cannot keep it
running or expose the local URL to their browser, give them the command to
run on their machine.

The user's decisions and notes save to `feedback.json` after selection or
leaving an edited field. Tell them to wait for **Saved to the audit folder**
before closing the page. When they ask you to continue, run `resume` again
and read `feedback` and `feedbackReview` before editing the report or proposing
next actions. Local saves do not automatically notify the agent in chat.

Failed saves leave the edits in the page and show **Retry saving**. For a conflict, use **Copy feedback JSON** to preserve unsaved edits, reload, and review the newer saved data before applying changes. Unsaved edits do not survive closing the page. After restarting the viewer, open its newly printed URL.

To import feedback from an older local report or an artifact export:

```sh
spectre audit import --project /path/to/my-project --feedback /path/to/feedback.json
```

Review the export's report title and commit against the intended audit first. Legacy exports have no stable audit identity. Identical imports do nothing. Conflicting records fail without changing the file. After reviewing both records, add `--replace` to replace conflicts. Import preserves unmatched records for review.

## Multiple audits and projects

```sh
spectre audit list
spectre audit resume --project /path/to/my-project --audit "<audit-id>"
```

`--audit` also selects an audit for checkpoint, view, and feedback import. Saving a checkpoint makes that audit active. Listing projects does not search the entire filesystem.

The discovery index is `~/.audit-report/projects.json`. Set `AUDIT_REPORT_HOME` to use another directory. Resume re-registers a project if its index entry is missing or its old location no longer exists. Copying a dotfile into a second live project is rejected because the two projects would have the same identity.

When moving a project, keep its audit folder and update `dataDir` in the dotfile if necessary. IDs must remain unchanged. Update `report.repoRoot` before making standalone exports. Back up the dotfile and sibling audit folder together; the local index can be rebuilt by resuming each project.

## Export and recovery

Build a standalone report from the `paths.report` returned by resume:

```sh
spectre audit build "/path/to/my-project.audits/<audit-id>/report.json" --strict
```

Standalone files keep browser-local feedback. Hosted artifacts keep their existing database feedback. Neither automatically synchronizes with the local audit folder.

New audit reports and `build --init` starters contain a `reportId`. Preserve it
when editing or moving the report. Use a new ID for a separate report.
For older JSON without this field, standalone builds use the absolute source
path as their storage identity and leave the source unchanged. Moving that
source changes its browser storage identity; export feedback before moving it.

If feedback exists under the older title-and-commit key, the page offers
**Import older feedback for this report**. Use it only after confirming that
the feedback belongs to this report. Import preserves current decisions,
drafts, and the original browser copy. It does not affect persistent viewer
feedback, which already belongs to an audit ID.

Do not edit `feedback.json` while a viewer or import is writing it. The commands use lock files and atomic replacement. After a hard process termination, an error may name a remaining lock and the owning process ID. Verify that process has exited before removing its lock. A viewer lock is `<audit-directory>/.viewer.lock`; write locks end in `.lock`. Never remove the underlying report or feedback file to resolve a lock.
