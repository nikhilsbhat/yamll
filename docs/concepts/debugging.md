# Debugging YAML

Yamll includes debugging commands for questions that usually become painful in large YAML repositories.

## Where Did This Value Come From?

Use `trace` when you know the final path and need the source file and line.

```bash
yamll trace service.yaml:metadata.labels.team
```

Example output:

```text
origin: common/base/service.yaml:12
```

## Why Did This Value Win?

Use `explain` when a value may have been imported, merged, and overridden.

```bash
yamll explain service.yaml:spec.template.spec.containers.0.image
```

`explain` shows the final value, the selected origin, and previous candidates that were overridden.

## What Depends on This File?

Use `impact` before editing shared YAML.

```bash
yamll impact common/base.yaml
```

This is the inverse of `tree`: it shows downstream files affected by a change.

## What Is Wrong With This Graph?

Use `lint` for import and merge hygiene.

```bash
yamll lint -f service.yaml
```

Lint checks include:

- Duplicate keys
- Unresolved imports
- Unused imports
- Circular references
- Invalid anchors
- Conflicting merges
- Duplicate libraries
- Dead imports
- Overridden imports
