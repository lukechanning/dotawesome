# Development

Install Go 1.26+ and Gitleaks. Build from the checkout with `bin/dotawesome`.
The launcher serializes builds, checks source/module fingerprints, builds both
executables in a staging directory, and atomically replaces successful artifacts.
A compilation failure never launches the previous binary. `.build/` is ignored.

| Location | Responsibility |
| --- | --- |
| `cmd/dotawesome` | Process entry point |
| `cmd/dotawesome-work` | Standalone installed Niri layout helper |
| `internal/app` | Typed inspection, validation, packages, shared setup and age services |
| `internal/deploy` | File ownership, atomic replacement, snapshots, locking and recovery |
| `internal/components` | Explicit component registry, TOML profiles and rendering |
| `internal/external` | Captured commands and terminal command boundaries |
| `internal/cli` | Explicit commands and PTY scenarios |
| `internal/tui` | Bubble Tea dashboard and workflow state machine |
| `home/`, `components/` | Maintained workstation configuration and pinned dependencies |
| `encryption/` | Public recipient and encrypted recovery copy only |

```sh
bin/ci                 # formatting, vet, race-enabled tests, redacted secret scan
bin/ci --integration   # additionally require and run external/native checks
```

Fast tests use temporary homes, fake system commands and real file transactions.
They need no desktop tools. PTY tests build a disposable binary and cover startup,
resize, persistent completion, external-command handoff, safe cancellation and
terminal restoration. Package tests preserve attribution boundaries; recovery tests
cover unexpected edits and failed rollback. No test deploys into the live home. Readiness and flow tests cover ordered setup,
reused enrollment, declined steps, independent app setup and managed-only sync.

Integration requires chezmoi, age, age-keygen, Fish, Niri, Noctalia, wlr-which-key,
systemd-analyze, Neovim and tmux. Missing prerequisites fail with a named tool.
Native checks stage generated files under `/tmp`.
The jump-menu check parses the generated menu and selects Escape through
`--initial-keys`, without opening an overlay or running a menu command.
Age tests generate disposable identities and verify passphrase enrollment,
recovery and repeat shared setup in
private homes. Sync tests use disposable local Git remotes to verify fast-forward
updates and refusal of dirty or divergent checkouts, without network access. tmux and systemd validation need local Unix-socket permission.
Neovim validation uses private XDG directories with no plugin downloads.

The Go module pins Bubble Tea, Bubbles and Lip Gloss v2, go-toml/v2, and the
creack/pty test helper. Standard library testing, flags, JSON, filesystem and
subprocess APIs cover the rest. No assertion framework or extra task runner.

Migration evidence (2026-10-19): the Python baseline ran 156 tests in 19.611 s
with local external tools. Temporary parity checks compared all built-in Niri
profiles plus Neovim/tmux against Python output, with only the intentional helper
binding difference. A real Python-generated ledger/snapshot was updated and
restored by Go, including original symlinks. The parity harness was then removed;
focused Go tests retain these guarantees. Python runtime modules, prompt tests,
retired configurations and historical planning documents were removed; Git
history remains intact. The warm-cache normal Go check completed in 5.703 s, including formatting,
vet, race tests, PTY binary builds and secret scanning. This is a local timing,
not a cross-machine benchmark. A full integration run completed in 5.543 s
with warm caches. At cutover, tests grouped behavior into 42 top-level cases
with table/PTY subcases, rather than 156 Python cases. The test categories are
transactions/ownership/recovery, component rendering, package attribution,
secrets/bootstrap, CLI/terminal lifecycle, UI navigation, and work-layout IPC.
Python used no third-party runtime packages; Go adds the three Charm libraries,
go-toml and a PTY test helper, plus Charm's transitive dependencies. The terminal
restoration assertions also use Charm's existing x/sys dependency.

Manual acceptance on both G14 and desktop remains required: log into the packaged
Niri session, verify display identities/scales and docking, wallet/portal behavior,
Noctalia locking/media controls, work-layout launches, and return to KDE. Test a
real terminal resize and sudo/login handoff on each machine. Automated checks do
not substitute for hardware or graphical-session acceptance.
