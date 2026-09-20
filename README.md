# pod-ssh

`pod-ssh` is a searchable terminal interface for opening an interactive shell
in Kubernetes pods. It is distributed as a standalone Go binary and does not
require Python, `gum`, or `kubectl`.

<p align="center">
  <img src="./pod-ssh.gif" alt="pod-ssh terminal demo">
</p>

The existing command syntax remains supported:

```bash
pod-ssh
pod-ssh api@production
pod-ssh api@production bash
pod-ssh history
pod-ssh last
pod-ssh last~2
```

## Features

- Searchable pod, container, and history pickers.
- Partial pod-name matching.
- Colored output, loading spinners, and keyboard navigation.
- Native Kubernetes API and interactive exec support.
- Standard `KUBECONFIG` loading and current-context behavior.
- Bash, Zsh, Fish, and PowerShell completion generation.
- History compatibility with earlier versions of `pod-ssh`.
- Native self-update support for release binaries.

## Development

The project uses [mise](https://mise.jdx.dev/) to pin its Go toolchain. Install
the tools and build the binary with:

```bash
mise install
mise run build
```

If `mise` is unavailable, install it with:

```bash
curl https://mise.run | sh
```

The binary is written to `bin/pod-ssh`.

Run the project-owned checks and smoke tests with:

```bash
mise run check
mise run smoke
mise run release-check
mise run release-snapshot
```

The check task applies `go fix`, formats the source, runs `go vet`, and executes
the test suite with the race detector.

Pushing a semantic version tag such as `v2.0.0` runs GoReleaser, publishes
platform archives and checksums to GitHub Releases, generates release notes,
and updates [CHANGELOG.md](CHANGELOG.md) on `main`.

## Usage

Running without a target opens searchable namespace and pod pickers. The
namespace configured by the current Kubernetes context is marked `current`:

```bash
pod-ssh
```

Type `/` to filter the pod list, use the arrow keys to navigate, and press Enter
to connect. If the selected pod has multiple containers, `pod-ssh` opens a
second searchable picker.

Use the compact `pod@namespace` syntax to select a namespace explicitly:

```bash
pod-ssh telemetry@monitoring
```

An exact pod name connects immediately. A unique partial name also connects
immediately; multiple matches open a searchable picker.

Choose a shell using the existing second positional argument:

```bash
pod-ssh api@production bash
```

If the requested shell is not installed in the container, `pod-ssh` tries the
other common shells (`sh`, `bash`, and `zsh`) before returning the original
exec error.

### History

```bash
pod-ssh history
pod-ssh last
pod-ssh last~3
pod-ssh clear-history
```

History remains in `~/.pod_ssh_history`, and the 20 most recent connections
remain in `~/.pod_ssh_last`.

### Kubeconfig

`pod-ssh` follows Kubernetes' standard loading rules:

1. Use the files listed by `KUBECONFIG` when it is set.
2. Otherwise use `~/.kube/config`.
3. Respect the current context and its default namespace.

An explicit file can be supplied when needed:

```bash
pod-ssh --kubeconfig ./cluster.yaml
```

Kubeconfig `exec` credential plugins are handled by the Kubernetes Go client.
The external credential command referenced by such a kubeconfig must still be
installed.

### Shell completion

Generate completion scripts with:

```bash
pod-ssh completion bash
pod-ssh completion zsh
pod-ssh completion fish
pod-ssh completion powershell
```

Completion includes commands, pods in the current namespace, and namespaces
after the `@` separator. Cluster-backed completion has a short timeout so an
unavailable cluster does not block the shell.

### Self-update

Release binaries can update themselves with:

```bash
pod-ssh self-update
```

The executable's directory must be writable by the current user.

## License

This project is available under the [MIT License](LICENSE).
