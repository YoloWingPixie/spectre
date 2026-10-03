# Specification format

The agent creates or maintains this folder for review in Spectre. It is
separate from the Spectre checkout. See [the operating skill](../SKILL.md)
for how to present it to the user and incorporate feedback.

```sh
spectre spec -dir /path/to/spec -port 0
```

`-dir` defaults to the current directory. `-port` defaults to `8765`; use `0`
to choose an available port. Spec files are re-read when they change, so edits
appear on the next page load. The server binds to `127.0.0.1` and rejects
foreign `Host` headers and cross-origin POSTs.

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
fold-in)" until the answer is incorporated into `open-decisions.md` and its
review entry is removed or archived. The agent can perform this step when
asked to update the specification. Preserve source feedback until it has
been incorporated; archive it according to the agreed review workflow.
