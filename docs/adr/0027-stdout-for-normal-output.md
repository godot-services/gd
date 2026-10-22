# 0027. Normal output on stdout

- Status: accepted
- Date: 2026-10-23
- Refs: [gd#75](https://github.com/godot-services/gd/issues/75), [gd#72](https://github.com/godot-services/gd/issues/72)

## Context

Commands print with cobra's `cmd.Printf`/`cmd.Println`. Cobra sends those to stderr unless `SetOut` is called, and `Execute` never called it. So the normal output of `gd add`, `install`, `remove`, `editor`, `templates`, `cache` and others went to stderr: `gd templates list | cat` showed nothing piped and `gd cache info 2>/dev/null` printed nothing. Tests did not notice because their helper calls `SetOut` itself.

## Decision

`newRootCmd` calls `SetOut(os.Stdout)`, so every `cmd.Print*` and `cmd.OutOrStdout()` writes to stdout. Errors, warnings and hints keep using `cmd.PrintErr*` and `ErrOrStderr()`, which stay on stderr. Informational results such as "already installed" or "no editors installed" count as normal output.

## Consequences

Output can be piped and filtered. Tests that override the writer with `SetOut` keep working. A new command must pick `PrintErr*` for anything that is not its result; a test runs a command without `SetOut` and fails if its output does not reach stdout.
