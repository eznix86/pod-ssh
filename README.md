# pod-ssh

`pod-ssh` is a command-line utility for connecting to Kubernetes pods.

<p align="center">
  <img src="./pod-ssh.gif">
</p>

## Installation

```bash
sudo curl -sSL \
  https://raw.githubusercontent.com/eznix86/pod-ssh/main/pod-ssh \
  -o /usr/local/bin/pod-ssh && \
sudo chmod +x /usr/local/bin/pod-ssh
```

Ensure `/usr/local/bin` is in your `PATH`.


Note: **You can keep it as `pod-ssh` but it may also rename the script to any name you like, for example to `kssh` or `pssh`** 

---

## Dependencies

The script will install these automatically if missing:

* [`gum`](https://github.com/charmbracelet/gum) – for interactive prompts and formatting
* [`kubectl`](https://kubernetes.io/docs/tasks/tools/) – to interact with Kubernetes pods

Supported platforms:

| Platform      | gum install         | kubectl install       |
| ------------- | ------------------- | --------------------- |
| macOS         | Homebrew            | Homebrew              |
| Arch Linux    | yay / paru / pacman | pacman                |
| Debian/Ubuntu | apt + wget/tar      | apt + curl            |
| Other Linux   | wget + tar          | curl (generic binary) |

---

## Usage

```bash
pod-ssh [COMMAND | POD@NAMESPACE]
```

### Commands

| Command                 | Description                                |
| ----------------------- | ------------------------------------------ |
| `pod-ssh`               | Interactive selection of namespace and pod |
| `pod-ssh help`          | Show help                                  |
| `pod-ssh history`       | Show interactive history selection         |
| `pod-ssh clear-history` | Clear connection history with confirmation |
| `pod-ssh last`          | Reconnect to the most recent pod           |
| `pod-ssh last~N`        | Reconnect to the Nth most recent pod       |

### Direct Connection

You can skip the interactive selection if you know the pod and namespace:

```bash
pod-ssh mypod@namespace
```

The script will:

* Automatic ssh when matching the pod name or offer fuzzy matches interactively.
    * `pod-ssh telemetry-app@telemetry-system` -> `pod-ssh telemetry-app-d4fd85895-r27tg@telemetry-system`
---

## History and State

`pod-ssh` maintains simple text files in your home directory:

| File                 | Purpose                                  |
| -------------------- | ---------------------------------------- |
| `~/.pod_ssh_history` | Stores unique pod@namespace combinations |
| `~/.pod_ssh_last`    | Stores the last 20 connections in order  |

These files enable the `history` and `last` commands for fast reconnection.

---

## Examples

Interactive pod selection:

```bash
pod-ssh
```

Reconnect to the last used pod:

```bash
pod-ssh last
```

Reconnect to the third most recent pod:

```bash
pod-ssh last~3
```

Clear your history:

```bash
pod-ssh clear-history
```

Show available commands:

```bash
pod-ssh help
```


## License

This project is open source and available under the [MIT License](LICENSE).


## Contributing

Contributions are welcome.
To propose changes:

1. Fork the repository
2. Create a new branch
3. Submit a pull request with a clear description of your changes
