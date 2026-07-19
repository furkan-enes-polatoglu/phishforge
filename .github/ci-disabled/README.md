# Enabling CI

`ci.yml` lives here (not under `.github/workflows/`) because the token used for the
initial push lacked the `workflow` scope. To enable GitHub Actions:

```bash
git mv .github/ci-disabled/ci.yml .github/workflows/ci.yml
git commit -m "ci: enable GitHub Actions"
git push
```

Pushing a file under `.github/workflows/` requires a token/SSH key with the
`workflow` scope (e.g. `gh auth refresh -s workflow`).
