# deploy-docker — docs

**Docker deploy.** Build the app image and run/push it (Dockerfile) to a host or registry.

## Install

```bash
togo install togo-framework/deploy-docker
```

Registers on the [`deploy`](https://github.com/togo-framework/deploy) base; select it with **deploy.provider in togo.yaml (or DEPLOY_PROVIDER)**, then use **`togo deploy`**.

## Interface

`Deployer` — `Provision`/`Deploy`/`Destroy`/`Status` over a `Spec{App,Dir,BuildCmd,Host,User,Image,Region,Domain}` built from your `togo.yaml`.

## Configuration

| Env var | Description |
|---|---|
| `DOCKER_REGISTRY` | Container registry to push the built image to (e.g. `ghcr.io/you`). Optional — runs locally if unset. |

## Usage & notes

Builds the image, optionally pushes to `DOCKER_REGISTRY`, and runs it on the target host over SSH (or locally). `Destroy`/`Status` use `docker inspect`.

## Example

```bash
togo deploy --provider docker --dry-run   # preview the plan
togo deploy --provider docker
```

## Links

- [Marketplace](https://to-go.dev/marketplace)
- [Source](https://github.com/togo-framework/deploy-docker)
