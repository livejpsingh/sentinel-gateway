# Release and CI/CD

## GitHub Actions

`.github/workflows/ci-cd.yml` runs CI for pull requests to `main`, pushes to
`main`, version tags beginning with `v`, and manual workflow dispatch.

The validation job runs on Go 1.22 and the current stable Go release:

1. `go mod download`
2. `go test -race ./...`
3. `go vet ./...`
4. `go build ./cmd/sentinel ./mocks/users ./mocks/payments`

The image-publishing job depends on successful validation and runs only for
pushes to `main` or `v*` tags. It builds the gateway target from
`deploy/Dockerfile` and publishes `ghcr.io/livejpsingh/sentinel-gateway` to
GitHub Container Registry using the repository `GITHUB_TOKEN` with package
write permission.

Tags include branch or version ref, commit SHA, and `latest` on the default
branch. A PR does not publish an image.

## Create a release

After CI is green on the intended commit:

```powershell
git checkout main
git pull --ff-only
git tag -a v1.0.0 -m "SENTINEL v1.0.0"
git push origin v1.0.0
```

Use the next appropriate semantic version for actual release content. Confirm
the Actions publish job succeeds, then verify the image tags/digest in GHCR.
The GitHub repository may need package write permission enabled for Actions.

## Delivery is not deployment

This workflow provides continuous delivery of a tested gateway image. It does
not deploy to a production host or cluster. No target platform, credentials,
approval policy, health checks, or rollback plan are defined in this project.
Production deployment should be a separate environment-protected workflow or
release system with reviewed credentials, readiness checks, and rollback.

## Container

The Dockerfile compiles a static Go binary and copies it into
`gcr.io/distroless/static-debian12:nonroot`. The mock services use the same
Dockerfile with a different `APP` build argument in Compose. The runtime has no
shell and runs as the non-root distroless user.

Do not place production secrets in Docker build arguments or the image's copied
`config/config.yaml`. Supply them only at runtime through an appropriate secret
mechanism. The checked-in default is intentionally only a local demo key.

## Release checklist

- [ ] Tests, race detector, vet, and build pass.
- [ ] Docs and environment table match the implementation.
- [ ] No secrets or local state are staged.
- [ ] Version tag points to the reviewed commit.
- [ ] GHCR image and tags are verified.
- [ ] Deployment environment runs smoke checks and has a rollback route.
