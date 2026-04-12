# Development Guidelines

Use `agg_go` as the primary rendering and math implementation wherever possible, with a strong preference over custom geometry/math code.

- For affine transforms (translation, rotation, scaling, inversion), prefer `agg_go` types and helpers (`Transformations`, `Translation`, `Rotation`, `Scaling`) and only add minimal glue when needed to map existing behavior.
- Avoid re-implementing transform logic manually.
- Use custom implementations only when `agg_go` explicitly does not provide an equivalent API for the required behavior.
