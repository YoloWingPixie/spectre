# ASOC role: acceptance scenarios

Each scenario becomes a textsim script (`cmd/textsim/testdata/asoc-<name>.txt`,
ASOC-NFR-TST-002) on the fake sim and fake clock. The full radio dialogues
are written out in `design.md` §9.4; the scenarios point at them and add
the checks the script asserts (`@expect`, `@expect-none`, `@must-not`,
`@asoc` for the `/asoc` dump, `@log`). Setup: the Nevada example of
`design.md` §9.1 unless stated.

---

## M1 scenarios

### S-01 · Identity, radio check, unidentified caller
Script `asoc-identity.txt`.

| T | Speaker | Transmission / event | Checks |
|---|---|---|---|
| 00:00 | voice A (264.0) | "Overlord, radio check." | |
| 00:01 | Overlord | "Station calling Overlord, say callsign." | `@expect "say callsign"`; `@asoc sessions=0` |
| 00:10 | voice A | "Overlord, Hawg 1-1, radio check." | |
| 00:11 | Overlord | "Hawg 1-1, Overlord, loud and clear." | voice bound; `@log "asoc: bound unit"` |
| 00:20 | voice B (red unit) | "Overlord, Flanker 1-1, checking in." | `@expect-none`; `@log "other coalition"` |
| 00:30 | voice A | "Hawg 1-2, Hawg 1-1, go button four." | `@expect-none` (chatter) |
| 00:40 | voice A | (STT empty) | `@expect "Hawg 1-1, say again"` once |

**Covers:** ASOC-FR-IDN-001, IDN-002, IDN-003, ROL-004, NLU-007, PHR-002, VOX-002.

## M2 scenarios

### S-06 · Tanker flow
Script `asoc-tanker.txt`. Dialogue `design.md` §9.4.3.

| Checks |
|---|
| `@expect "Texaco one, AR track Shell, flight level two two zero to two four zero, boom"` |
| Eagle 2 → `@expect "number two for Texaco one"` with FL240–260 |
| an F/A-18 flight → `@expect "negative, no tanker on station"` (boom only) |
| off the tanker → re-cleared into 88AY |
| no SRS client on 276.1 |

**Covers:** ASOC-FR-AR-001, AR-002, AR-003, AR-004, AR-005, AR-006.

## M3 scenario

### S-09 · Gateway out and in with ATC
Script `asoc-gateway.txt` (`-role atc,asoc`). Dialogue `design.md` §9.4.4.

| Checks |
|---|
| Nellis Clearance: `@expect "cleared to gateway Coyote"` and `@expect "expect due regard at Coyote"` |
| Los Angeles Center at Coyote: `@expect "contact Overlord, two six four point zero"`; `@log "asoc: handoff accepted"` |
| with the ASOC absent the same release says "change to Darkstar frequency approved" (golden) |
| RTB: `@expect "at Coyote contact Los Angeles Center, one three three point niner five"`; the facility's mailbox holds the hand-off |
| Los Angeles Center gives the pop-up IFR return with the code kept |

**Covers:** ASOC-FR-GTW-001, GTW-002, GTW-003, GTW-004, GTW-005, GTW-006, GTW-008.

---

## M4 scenarios

### S-10 · Co-located Darkstar on two frequencies; ground coverage
Script `asoc-platforms.txt`. Part 1: `asoc = {{ freq = 264.0, platform = "awacs", awacs = "Darkstar" }}`. Part 2: ground CRC-1, `sensor.mode = "realistic"`.

| Checks |
|---|
| part 1: "Darkstar, Hawg 1-1, checking in" on 264.0 → "Hawg 1-1, Darkstar, radar contact, …" (ASOC); on 251.0 the AWACS answers as today |
| part 1: voice of the ASOC ≠ Darkstar's (`@log "voicealloc: asoc"`) |
| part 1: identifying on 264.0 does not identify the voice on 251.0 |
| part 2: Hawg 1 at 1,000 ft AGL behind the ridge: "not radar contact"; at 15,000 ft: "radar contact" |
| part 2: CRC-1 destroyed → `@log "asoc: platform lost"`, no further ASOC transmission |

**Covers:** ASOC-FR-ROL-005, ROL-006, ROL-008, VOX-003, COV-001, COV-003, COV-004, COV-005, CHK-007.
