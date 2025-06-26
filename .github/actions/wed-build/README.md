# Build with Wednesday

Build a static site using the [`wed`](https://github.com/DazFather/Wednesday) CLI — a lightweight frontend framework designed for simplicity and power.

This GitHub Action downloads the `wed` binary (for Linux, macOS, or Windows), sets it up in the CI environment, and runs `wed build` with given flags if passed.

## Usage

```yaml
uses: ./.github/actions/wed-build
with:
  version: v1.2.3                              # Optional. Defaults to the latest release.
  flags: --settings my/path/to/settings.json   # Optional. Flags to pass to `wed build`
````

### Example in a full Pages deployment:

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/configure-pages@v5
      - uses: DazFather/Wednesday/.github/actions/wed-build@v.1.0.0-alpha.8
      - uses: actions/upload-pages-artifact@v3
        with:
          path: './build'

      - uses: actions/deploy-pages@v4
        id: deployment
```

---

## Inputs

| Name      | Description                                | Default  |
| --------- | ------------------------------------------ | -------- |
| `version` | The release tag to install (e.g. `v1.2.3`) | *Latest* |
| `flags`   | Flags passed to the `wed build` command    | `""`     |

---

## Requirements

* Your project must include a valid Wednesday project (e.g. created with `wed init`)
* The output directory defaults to the same as `wed build` (`dist/`, unless overridden)

---

## Links

* 📦 [Wednesday CLI](https://github.com/DazFather/Wednesday)
* 🚀 [GitHub Pages Deploy Action](https://github.com/actions/deploy-pages)


