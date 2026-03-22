# TypeScript Build Flow

1. `npm ci` — install deps (clean)
2. `npx tsc --noEmit` — type check
3. `npx eslint .` — lint
4. `npm test` — run tests
5. `npm run build` — production build
