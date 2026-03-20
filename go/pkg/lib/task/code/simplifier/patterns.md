# Simplification Patterns

## Consolidate
- Three similar blocks → extract helper function
- Duplicate error handling → shared handler
- Repeated string → constant

## Flatten
- Nested if/else → early return
- Deep indentation → guard clauses
- Long switch → map lookup

## Remove
- Unused variables, imports, functions
- Wrapper functions that just delegate
- Dead branches (always true/false conditions)
- Comments that restate the code

## Tighten
- Long function (>50 lines) → split
- God struct → focused types
- Mixed concerns → separate files
