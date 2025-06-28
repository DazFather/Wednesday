# Build with Wednesday

Build a static site using the [`wed`](https://github.com/DazFather/Wednesday) CLI — a lightweight frontend framework designed for simplicity and power.

This GitHub Action downloads the `wed` binary (for Linux, macOS, or Windows), sets it up in the CI environment, and runs `wed build` with given flags if passed.

## Usage

```yaml
uses: DazFather/Wednesday/wed-build@v1.0.0-alpha.8
with:
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
      - uses: DazFather/Wednesday/wed-build@v1.0.0-alpha.8
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: './build'
      - uses: actions/deploy-pages@v4
        id: deployment
```

---

## Inputs

The only allowed input is `flags` that allows you to pass whatever you need to the wed build command,
you might want to use it if your project settings file is not on the root directory or has a non-default name.
The input is totally optional.

---

## Requirements

* Your project must include a valid Wednesday project (e.g. created with `wed init`)
* The output directory defaults to the same as `wed build` (`dist/`, unless overridden)

---

## Links

* 📦 [Wednesday CLI](https://github.com/DazFather/Wednesday)
* 🚀 [GitHub Pages Deploy Action](https://github.com/actions/deploy-pages)


