# Command Reference

This section links to the generated Cobra command reference.

| Command | Purpose |
|---------|---------|
| [yamll](doc/yamll.md) | Root command and global flags |
| [yamll build](doc/yamll_build.md) | Render YAML with imports resolved and anchors expanded |
| [yamll import](doc/yamll_import.md) | Import YAML libraries and output a combined stream |
| [yamll tree](doc/yamll_tree.md) | Print the dependency graph as text, JSON, DOT, or Mermaid |
| [yamll trace](doc/yamll_trace.md) | Map a rendered path back to its source file and line |
| [yamll explain](doc/yamll_explain.md) | Explain the final value and merge resolution for a path |
| [yamll lint](doc/yamll_lint.md) | Detect dependency, anchor, and merge issues |
| [yamll impact](doc/yamll_impact.md) | Show downstream files affected by a dependency |
| [yamll lock](doc/yamll_lock.md) | Generate a lock file for reproducible imports |
| [yamll lock verify](doc/yamll_lock_verify.md) | Verify resolved imports against the lock file |
| [yamll lock explain](doc/yamll_lock_explain.md) | Explain which roots pull in a dependency |
| [yamll version](doc/yamll_version.md) | Print the installed Yamll version |
