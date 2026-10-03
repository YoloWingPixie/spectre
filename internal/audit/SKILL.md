---
name: spectre-audit
description: Write or resume evidence-based codebase audits with persistent project checkpoints, findings, and reader decisions. Build an interactive HTML report from report.json for local viewing, sharing, or a claude.ai artifact. Use when asked to audit, review, assess, or survey a codebase and produce or continue a report.
---

# Spectre audit mode

Turns a `report.json` into one HTML page with severity and status filters, search, a "Fix first" list, per-area findings, and editor links for every file reference.

Tool location: the folder that holds this file. Below, `$AR` means that folder.

The developer runs their coding agent in the project being audited. Keep that project as the working directory; `$AR` holds the extracted schemas and instructions. Use `spectre audit` from PATH. The executable contains the audit engine; Node, Task, and npm are not needed for audits. Git supplies repository fingerprints. On Windows, run inside WSL. If the agent's file-access controls request access to the tool or sibling audit folder, use the client's normal approval flow.

## When to use

- Someone asks for an audit, review or assessment and wants a report they can read, share or act on.
- You have findings that need proof, a fair view of the current design, and options to choose from.

Do not use it for a quick answer in chat. Just answer.

## Workflow

1. **Resume first.** Check the audited repository root for `.audit-report.json`. If present, run `spectre audit resume --project /path/to/project`. Read the checkpoint, report, and feedback before continuing. If absent, use `init` with the same `--project`. Add `--new` only for an explicitly separate audit. For persistence commands, checkpoint input, imports, and recovery, read `$AR/docs/resuming-audits.md`.
2. **Investigate.** Read the code. Run the app or commands if you can. If `repositoryChanged` is true or unknown, revalidate affected evidence instead of assuming the checkpoint still proves it.
3. **Verify each finding.** Open the file and read the lines you cite. Run the command you quote. Drop anything you cannot confirm, or move it to `openQuestions`.
4. **Write and checkpoint.** Edit the `paths.report` returned by resume. Preserve finding and option IDs. Field reference: `$AR/schema/report.schema.json`. Save a checkpoint after meaningful work and before handing control back; record completed work and next actions. Use the latest `projectRevision` to prevent overwriting another session. Read saved decisions as feedback, not authorization to change code.
5. **Build.** `spectre audit build /path/from/resume/report.json --strict`. Fix every error and warning. Errors list the path to each bad field.
6. **Look once.** Open the HTML (or screenshot it) at desktop and phone width. Fix what is visibly wrong. Do not loop.
7. **Share.** For automatic local feedback, run `spectre audit view --project /path/to/project` in a persistent terminal session and give the printed URL. If the agent cannot keep a process running after its turn, give the developer that command to run in a second terminal. Otherwise give the built HTML path. To publish as a claude.ai artifact, publish the HTML file as is. It already meets the artifact page rules.

## Evidence standards

- Only claims you checked. If you did not read it, do not cite it.
- `path` is relative to `repoRoot`. Give the exact `line` (and `endLine` for a range). Links are built from these, so a wrong line is a broken link.
- Excerpts are short (15 lines or fewer) and copied exactly. Use `...` for cut lines.
- Each file item gets a one-sentence `note` that says what the lines prove.
- Use `command` evidence for greps, test runs and queries. Paste the real output. An empty string means "no output", which is often the proof.
- Use `screenshot` evidence for UI problems. The path is relative to the JSON file.
- Show the good pattern too when the codebase already has one. It makes the fix obvious.

## "How it is now" (current)

Be fair. The current design usually exists for a reason.

- `description`: what the code does today, in plain words.
- `pros`: the real reasons it is built this way (simpler, faster to ship, matches a spec).
- `cons`: what breaks, and for whom.
- For a fixed finding, describe what the audit found, and say so ("At commit `abc123`, ...").

## Courses of action (coas)

- Give 2 to 4 options that are really different. Not three sizes of the same fix.
- Include "Do nothing" (accept the risk) when that is a reasonable choice. Say what that costs.
- Each option: `name`, `description`, `pros`, `cons`, `effort` (S/M/L), `risk` (low/med/high), and `touches` (files it changes) when you know them.
- Mark exactly one `recommended: true` with a one-sentence `recommendedReason`. Two recommended options fail the build.
- Every option has a cost. Empty `cons` is a warning.

## Severity

| Level | Use when |
|---|---|
| critical | Data loss, account takeover, or a full outage that one person can trigger now. |
| high | Serious harm that is likely, or needs only a small step: leaks across tenants, a server-wide hang, a core feature unusable for a group of users. |
| medium | Real harm, but limited in reach, or needs unusual conditions. |
| low | Small or rare harm, or a code-health problem that will cause bugs later. |

Rate by harm times likelihood today, not by how hard the fix is.

## Status

- `open`: not dealt with. Counted in the summary.
- `fixed`: the code changed. Add `statusNote` saying where ("Fixed in migration 0114"). Point evidence at the fixed code.
- `accepted`: the owner chose to live with it. `statusNote` says who and why.
- `wont-fix`: will not be fixed, for example the feature is being removed.

## Plain English (required)

Write so a busy developer who is new to the project understands on first read.

- Short sentences. One idea each. The build warns over 30 words.
- Common words. "Use", not "leverage" or "utilize". The build warns on a list of these.
- Explain any jargon or acronym the first time: "RLS (row-level security)".
- Say what breaks, for whom: "Night users cannot see links", not "Contrast issues exist".
- No hedging filler ("it is worth noting", "arguably") and no sales words ("robust", "seamless").
- Put code names in `backticks`.

| Don't | Do |
|---|---|
| Leverage a robust mechanism in order to facilitate throttling. | Check the rate limit before opening the transaction. |
| It is worth noting that pool exhaustion may potentially occur under load. | Ten reads at once use every connection. The server then hangs. |
| Suboptimal contrast ratios in dark mode. | Links are navy on near-black (about 2:1). Night users cannot see them. |
| RLS bypass in tests. | Tests run as the database superuser, so RLS (row-level security) is skipped. A broken policy still passes. |

More examples: `$AR/docs/writing-guide.md`.

## Editor links

Paths must be repo-relative. The page builds links from `repoRoot` + `path` + `line`, and the reader picks their editor in the header. A backticked `path/file.ts:12` in any text also becomes a link. In hosted viewers such as claude.ai artifacts the editor links may do nothing; the copy button next to each reference still works.

## Collecting the reader's feedback

Each finding has a "Your decision" box. The reader picks one COA, or "None of these" with a required reason. They can add a note on each COA and a general note. Every field saves by itself.

**Local persistent viewer.** The `view` command saves feedback in the sibling audit folder. On each resume, read `feedback` and `feedbackReview`. Preserve notes for removed findings or options until the reader resolves them. No Claude artifact capabilities are needed for this mode. A reader's saved decision does not authorize implementation by itself.

**Publish with feedback on.** Publish the HTML as an artifact with capabilities `{"db": {}, "comments": {}, "user": {}}` (the build prints this line). Such an artifact is internal to the organization and cannot be shared publicly.

**What arrives.** After you publish, you are watching the artifact. Each time a reader changes a field (picks an option, or leaves a text box they edited), the page saves it and posts one comment to a thread on that finding, sent to you. Nothing is sent if the content matches what was last sent. It arrives as a turn headed `[Artifact comment sent to Claude]`. The text looks like:

```
Audit feedback · LOG-01 · Ten REST reads at once can freeze the whole web server
Decision: none — We plan to replace the pool with a proxy next month.
Note on Check the limit before the transaction: Worth doing anyway if the proxy slips.
General note: Ask ops first.
Full feedback: db feedback/LOG-01
```

Later changes to the same finding reply in the same thread. Long text is cut with "…"; the db record always has all of it.

**What to do with each one.**
1. Read the comment. Read the full record with the ArtifactData tool: `get` on `feedback/<id>`, or `list` on collection `feedback`.
2. Reply briefly in that thread with the ArtifactComments tool. Say you got it and what you will do.
3. If the reader chose "None of these", ask one clear question, or propose a new COA.
4. Do not change code because of a decision unless the reader's note asks for it or the user told you to act on decisions.

**At the end.** Read everything with ArtifactData `list` on collection `feedback`. Each record is `{findingId, decision: {type: "coa", coaId} | {type: "none", reason} | null, coaNotes: {<coaId>: text}, note, updatedAt, updatedBy, notify}`. `coaId` matches `coas[].id` in your report.json, so keep COA ids stable when you edit.

**Updating the report.** Change statuses (for example to `fixed` with a `statusNote`), rebuild, and republish to the same URL. The saved feedback stays in the db.

**Local file.** Opened from disk, the page saves feedback in that browser only and says so. The reader clicks "Copy feedback JSON" in the header and pastes the result to you. It has the same records under `findings.<id>`.

**When Claude is not notified.** Only editors of the artifact can send to Claude, and only while a Claude session is watching. Otherwise the finding shows "Saved · not sent to Claude: <reason>" (for example "no Claude session watching" or "consent declined"), and the browser console logs the error code. Then read the feedback with ArtifactData.
