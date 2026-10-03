---
name: spectre
description: Prepare and present specifications or codebase audit reports to a user with Spectre, then read their saved feedback to continue the work. Use when presenting a specification or audit for browser review, or continuing work from saved Spectre feedback.
---

# Present a review with Spectre

You operate Spectre; the user reviews the result. Prepare the source content,
run the viewer, present its URL, and read the user's saved feedback when they
ask you to continue. Spectre supplies the browser interface and storage; you
supply the analysis and any requested changes to source files.

Use the user's chosen project and scope. Run `spectre` from PATH or use the
absolute path to the checkout's `bin/spectre`. Installed product commands need
no Node runtime or Task. Building from this checkout uses `task build`.

## Choose the mode

- **Spec:** present requirements, design, scenarios, tasks, and open decisions.
  Use the spec workflow below.
- **Audit:** present verified findings, evidence, and courses of action.
  Read [the audit skill](docs/audit/SKILL.md) for the investigation,
  report format, checkpoints, and feedback workflow. The agent investigates
  and writes findings; `audit init` starts with an empty report.

Keep the audited project as the working directory for audit work. References
in this skill resolve relative to the Spectre checkout, not that project.

## Spec workflow

1. Read an existing specification and its `review/` files before changing it.
   If creating a specification, use the six-file format in
   [the specification format](docs/spec-format.md). Preserve existing item IDs
   and prefix declarations so feedback and cross-references stay attached to
   their items.
2. Prepare the requested content in `README.md`, `requirements.md`,
   `design.md`, `scenarios.md`, `tasks.md`, and `open-decisions.md` within
   the spec folder. The spec's README is separate from this tool's README.
3. Start the viewer in a persistent terminal:

   ```sh
   spectre spec -dir /absolute/path/to/spec -port 0
   ```

4. Check that the printed URL loads. Give the user that exact URL and identify
   the chapters or decisions that need review. Explain that they can record
   verdicts, comments, and open-decision answers in the page.
5. When the user asks you to continue, read `<spec>/review/feedback.md`,
   `comments.md`, and `answers.md`. Missing files mean no feedback of that
   kind yet. The latest verdict for an item is current; earlier entries
   remain history. An answer marked pending fold-in has not yet been
   incorporated into the source specification.
6. Summarize the feedback and make the changes covered by the user's
   instructions. Reopen or reload the affected pages for another review.
   Preserve source feedback until it has been incorporated.

## Presenting the review

Keep the viewer process running while the user reviews. URLs bind to
`127.0.0.1` on the machine running Spectre. If your execution environment is
not reachable from the user's browser, explain the access requirement. If you
cannot keep a process running, provide the exact command for the user to run
on their machine. Audit mode also supports standalone HTML delivery.

For a local review, hand off three things: the working URL, what needs the
user's attention, and how to continue after they have saved their feedback.
Local feedback does not automatically create a new agent turn. Read the files
or run `spectre audit resume` when the user returns.

Use saved decisions within the user's existing authorization. A choice in
the viewer alone does not authorize implementation or publication. Follow the
user's requested delivery method for any standalone or hosted audit report.
