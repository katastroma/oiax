# Oiax

Platform bootstrap and teardown CLI for
[katastroma](https://github.com/katastroma).

## TODO

- CRD install, Establish wait, and root Application bootstrap are stubbed in
  `cmd/` but not yet integrated into a formal CLI. Pending design decisions on
  whether the root Application is still needed given the architectural pivot
  toward webhook-driven GitOps without CRDs as the primary loop.
