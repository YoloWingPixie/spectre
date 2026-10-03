# ATC role: open decisions

Every question the design docs leave open, plus the new ones this spec
raised. Each has options, a recommendation (what the requirements assume
today), the requirements it **blocks** (their status is `open → OD-nnn`)
and those it only **affects**. Sources: [D §16] atc-design, [P §12]
atc-procedures, [T §9] atc-terminal-procedures, [G §5] atc-gaps; "new"
marks a question raised by this spec.

All decisions were resolved on 2026-10-01 by accepting the recommendations; the "Blocks" lines are kept for traceability.

---

### OD-001 · CV NATOPS provenance (resolved)
- **Source:** [D §16 Q1]
- **Question:** The only complete public copy is the 2009 edition (Public Intelligence, Distribution Statement C). The user believes an older edition was FOIA-released. Which edition do we cite?
- **Options:** (a) cite the 2009 edition as "publicly posted"; (b) cite the FOIA edition once the user supplies it; (c) cite only secondary sources.
- **Recommendation:** (a) now, switch to (b) when the source is confirmed; procedures used are stable across editions.
- **Blocks:** –
- **Affects:** every CVC/MSH/CV1/CV3/CVD requirement (citations only).
- **Resolution:** Resolved 2026-10-01: recommendation accepted.

### OD-002 · DCS's own ATC (resolved)
- **Source:** [D §16 Q2]
- **Question:** Silence DCS's AI ATC at staffed airbases or let both run?
- **Options:** (a) leave both; (b) silence per staffed airbase via the scripting API (verify it exists); (c) mission flag.
- **Recommendation:** (c): `silence_dcs_atc` mission flag, default off, implemented only if the API is verified.
- **Blocks:** ATC-FR-FAC-018
- **Affects:** –
- **Resolution:** Resolved 2026-10-01: recommendation accepted.

### OD-003 · Red-owned fields (resolved)
- **Source:** [D §16 Q3]
- **Question:** ATC for red fields needs a second SRS client set. Worth it?
- **Options:** (a) player coalitions only; (b) a second client set per coalition.
- **Recommendation:** (a) by default; (b) only when a mission has red player slots and lists `coalitions = { "red" }`.
- **Blocks:** ATC-FR-FAC-017
- **Affects:** ATC-FR-FAC-001.
- **Resolution:** Resolved 2026-10-01: recommendation accepted.

### OD-004 · Tower frequencies from `radio.lua` (resolved)
- **Source:** [D §11] (new as a dependency)
- **Question:** `mappack.ParseRadio` reads IDs/callsigns only; the registry needs the VHF/UHF/HF frequencies. Who adds it and when?
- **Options:** (a) the mappack owner adds it (additive); (b) ATC parses `radio.lua` itself.
- **Recommendation:** (a), scheduled as T-M0-12; (b) only as a stop-gap behind a TODO if the owner is unavailable.
- **Blocks:** –
- **Affects:** ATC-FR-FAC-001, FAC-002.
- **Resolution:** Resolved 2026-10-01: recommendation accepted.

### OD-008 · AI traffic in advisories (resolved)
- **Source:** [D §16 Q6]
- **Question:** Include AI and other-coalition aircraft in traffic advisories?
- **Options:** (a) players only; (b) players + friendly AI; (c) everything DCS sees (filtered by sensor mode).
- **Recommendation:** (c) filtered by the sensor mode, never hostile aircraft inside hostile airspace; advisories only when they meet the proximity rule.
- **Resolution:** resolved as recommended, (c) (user review 2026-09-29): neutral and friendly AI are called as traffic, enemy aircraft inside friendly airspace as "unknown traffic", nothing inside hostile airspace; specified in the AI traffic area (ATC-FR-AIT-001…014).
- **Blocks:** –
- **Affects:** ATC-FR-FFL-002, SAF-002, ATC-FR-RAD-015.

### OD-076 · Civil cap scaling defaults (resolved)
- **Source:** new ([CT §7.3])
- **Question:** Should the default civil cap follow the script (a fixed 40, running from mission start) or the players?
- **Options:**
  - (a) the script: a fixed 40 from mission start;
  - (b) a fixed 40, but dormant until the first player connects (`idle` = 0);
  - (c) `per_player` = 8 with `idle` = 0;

  Scaling by server load (FPS) is not part of any option: it is P3, later and only if needed (ATC-FR-CIVT-032).
- **Recommendation:** (b).
- **Blocks:** ATC-FR-CIVT-003
- **Affects:** ATC-FR-CIVT-019.
- **Resolution:** Resolved 2026-10-01: recommendation accepted.
