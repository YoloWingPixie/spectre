# Writing guide

The reader is a developer who has ten minutes and did not do the audit. Every field should make sense to them on first read.

## Rules

1. One idea per sentence. Aim for 15 to 20 words. The build warns over 30.
2. Use the common word. See the table below.
3. Explain a term or acronym the first time it appears: "RLS (row-level security)", "the pool (the shared set of database connections)".
4. Say what breaks and for whom. Name the user, the admin, the tenant or the operator.
5. Give numbers when you have them: "10 connections", "about 2:1 contrast", "up to 25,000 members".
6. State facts. Drop "it seems", "arguably", "it is worth noting". If you are not sure, put it in `openQuestions`.
7. No sales words. Nothing is "robust", "seamless" or "best-in-class".
8. Put code names in `backticks`. The page shows them as code, and `path/file.ts:12` becomes a link.

## Words the build warns about

| Instead of | Write |
|---|---|
| leverage, utilize | use |
| in order to | to |
| facilitate | help, allow |
| streamline | simplify, speed up |
| robust | say what it survives: "retries three times" |
| seamless, holistic, synergy | drop it |
| it is worth noting, it should be noted, needless to say | drop it and state the fact |
| going forward | from now on, or drop it |
| paradigm | model, approach |
| empower | let, allow |
| delve | look |

## Field by field

**title**: what is wrong, as a statement.
- Don't: "Rate limiting concerns"
- Do: "Ten REST reads at once can freeze the whole web server"

**summary**: 1 to 2 sentences. What is wrong, then who it hurts.
- Don't: "There may be potential issues with connection handling under certain load conditions."
- Do: "Each REST read holds one database connection and asks for a second. With ten reads at once, every request hangs, including sign-in."

**evidence note**: one sentence on what these lines prove.
- Don't: "See code."
- Do: "`withRateLimit` takes connection 2 from the same shared pool."

**current.pros**: the honest reasons for today's design.
- Don't: "None." (almost never true)
- Do: "One helper wraps auth, the transaction and the rate limit, so most routes stay short."

**current.cons**: concrete harm.
- Don't: "Not ideal for scalability."
- Do: "Failed writes hand back their token, so someone probing the API is never slowed down."

**coa.name**: a short verb phrase.
- Don't: "Option 1 (preferred approach)"
- Do: "Check the limit before the transaction"

**coa.description**: what you would change, in one to three sentences.

**recommendedReason**: one sentence that compares.
- Don't: "This is the best option."
- Do: "It removes the cause, not just the symptom, and the safe pattern already exists in the codebase."

**statusNote**: where and how it was closed.
- Do: "Fixed: unique index migration 0114 + Manage Server proof."
- Do: "Accepted by the owner on 2026-07-02: wiki admins are trusted."

## Good COA sets

- Fix the cause / patch the symptom / do nothing.
- Change the code / change the config / change the process.
- Small safe fix now / larger redesign later.

Bad sets: three versions of the same fix; an obviously silly option added to make the pick look good.
