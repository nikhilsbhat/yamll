# Lock Files

Remote imports are powerful, but they can drift. Branches move, tags can be recreated, URLs can return different content, and OCI artifacts can be updated. A `yamll.lock` file records what Yamll resolved so future builds can reproduce the same output.

## Generate a Lock

```bash
yamll lock -f app.yaml
```

The lock file records resolved dependencies, checksums, and Git commit SHAs when the import comes from Git.

## Use the Lock

Build, import, tree, and trace commands use the lock file automatically when one is present.

```bash
yamll build -f app.yaml
yamll tree -f app.yaml
yamll trace app.yaml:service.port
```

## Verify in CI

Use lock verification when you want a fast reproducibility check without rendering the full output.

```bash
yamll lock verify -f app.yaml
```

If a dependency changes without updating the lock file, the command fails.

## Update the Lock

Regenerate the lock after intentionally changing imports or accepting new remote content.

```bash
yamll lock -f app.yaml
```

Commit `yamll.lock` with the YAML files that depend on it.

## Multiple Root Files

Pass every root that should be represented by the lock:

```bash
yamll lock -f api.yaml -f jobs.yaml -f ingress.yaml
```

The lock remains dependency-oriented, while `yamll lock explain` can show which roots pull in a dependency.

```bash
yamll lock explain common/base.yaml -f api.yaml -f jobs.yaml
```

For the full schema and current limitations, see the [lock file reference](LOCKFILE.md).
