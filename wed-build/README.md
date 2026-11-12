# Build with Wednesday

Build a static site using the [`wed`](https://github.com/DazFather/Wednesday) CLI — a lightweight frontend framework designed for simplicity and power.

This GitHub Action downloads the `wed` release binary for the current runner (Linux, macOS, or Windows), makes it available in the environment, and runs `wed build`.

> **Note:** this action currently downloads the pinned `wed` release `v1.0.0-alpha.9`.

## Usage

```yaml
uses: DazFather/Wednesday/wed-build@main
with:
  flags: --settings my/path/to/settings.json   # Optional. Flags to pass to `wed build`
```

### Example in a full Pages deployment

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
      - uses: DazFather/Wednesday/wed-build@main
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: './build'
      - uses: actions/deploy-pages@v4
        id: deployment
```

---

## Inputs

| Name    | Required | Default | Description |
|---------|----------|---------|-------------|
| `flags` | No       | `""`    | Flags passed to `wed build` (for example `--settings my/path/to/settings.json` or `--mini`). |

---

## Requirements

* Your repository must contain a valid Wednesday project (e.g. created with `wed init`).
* The build output defaults to `wed build`’s output directory (`build/`, unless overridden by your settings).

---

## Links

* [Wednesday CLI](https://github.com/DazFather/Wednesday)
* [GitHub Pages Deploy Action](https://github.com/actions/deploy-pages)


