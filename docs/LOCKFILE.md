# Lock File (yamll.lock)

`yamll` can resolve YAML imports from remote sources (git, URLs) and from file patterns. Remote sources are powerful but can drift over time. A lock file makes these imports reproducible by recording resolved versions and checksums.

This document describes how `yamll.lock` works today.

## Quick Start

Generate a lock file:

```sh
yamll lock -f path/to/root.yaml
```

Use the lock automatically (default behavior):

```sh
yamll import -f path/to/root.yaml
yamll build  -f path/to/root.yaml
yamll tree   -f path/to/root.yaml
yamll trace  path/to/root.yaml:some.path
```

Ignore the lock file for a single run:

```sh
yamll import -f path/to/root.yaml --no-lock
```

Use a non-default lock file path:

```sh
yamll import -f path/to/root.yaml --lock-file /path/to/yamll.lock
```

## What Gets Locked

`yamll lock` resolves the full dependency graph starting from the root file(s) provided via `-f/--file`.

For each resolved dependency, it records:

- `sha256`: checksum of the resolved YAML content.
- `git_commit`: the exact commit SHA used for git imports (when applicable).
- `constraint`: the ref that was requested in the git import (tag/branch/sha), when it can be inferred from the import string.

Pattern imports (`*.yaml`) are expanded and the lock includes a checksum for each matched file as well as the aggregated pattern node.

## How Reproducibility Works

On `import/build/tree/trace`, `yamll` attempts to load the lock file (default `yamll.lock`).

When a dependency is a git import, and there is a matching lock entry, the import is rewritten to pin it to the locked commit:

- input: `git+https://host/org/repo@v1.2.3?path=base.yaml`
- locked: `git+https://host/org/repo@<commitSHA>?path=base.yaml`

This ensures the same content is fetched even if `v1.2.3` is a moving tag or a branch.

If no lock file is found (or `--no-lock` is set), `yamll` behaves as it did previously.

## Import Shorthand

In addition to explicit `git+https://...` imports, `yamll` supports a shorthand format:

```
<host>/<org>/<repo>//path/to/file.yaml@ref
```

Examples:

- `github.com/org/repo//base.yaml@v1.2.0`
- `gitlab.com/org/repo//base.yaml@main`

These shorthand imports are normalized to canonical `git+https://...` imports before resolution and locking.

## Lock File Format

The lock file is YAML.

Top-level fields:

- `version`: lock schema version.
- `generated_at`: UTC timestamp when the lock was generated.
- `roots`: root input files passed to `yamll lock`.
- `entries`: list of resolved dependency entries.

Each entry may include:

- `type`: one of `file`, `pattern`, `http`, `git+`.
- `source`: the original dependency string used as the key for matching during subsequent runs.
- `constraint`: requested ref for git imports (if present).
- `resolved`: resolved location (file path or URL), if applicable.
- `git_commit`: resolved commit SHA for git imports.
- `sha256`: checksum of the resolved content.

## Limitations (Current)

- Lock matching uses the exact `source` string. If you change import strings in your YAML files, you should regenerate the lock.
- URL imports are checksummed, but are not automatically pinned unless the URL itself is versioned.
- The current implementation focuses on reproducibility for git imports by pinning to commit SHA.

