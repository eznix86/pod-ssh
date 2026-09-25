// Package cmd implements the pod-ssh command line.
package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/eznix86/pod-ssh/internal/history"
	"github.com/eznix86/pod-ssh/internal/kube"
	"github.com/eznix86/pod-ssh/internal/target"
	"github.com/eznix86/pod-ssh/internal/ui"
	updater "github.com/eznix86/pod-ssh/internal/update"
)

// Version is injected at build time.
var Version = "dev"

type options struct {
	kubeconfig string
	input      io.Reader
	output     io.Writer
	errOutput  io.Writer
	store      *history.Store
}

// Execute runs the root command.
func Execute() error {
	store, err := history.New()
	if err != nil {
		return err
	}
	root := newRootCommand(&options{
		input:     os.Stdin,
		output:    os.Stdout,
		errOutput: os.Stderr,
		store:     store,
	})
	return root.Execute()
}

func newRootCommand(options *options) *cobra.Command {
	root := &cobra.Command{
		Use:           "pod-ssh [POD@NAMESPACE] [SHELL | COMMAND...]",
		Short:         "Connect to Kubernetes pods quickly",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) > 0 && strings.HasPrefix(args[0], "last~") {
				return runLast(command.Context(), options, strings.TrimPrefix(args[0], "last~"))
			}
			targetValue := ""
			if len(args) > 0 {
				targetValue = args[0]
				args = args[1:]
			}
			return runConnection(command.Context(), options, targetValue, args)
		},
	}
	root.Flags().SetInterspersed(false)
	root.SetIn(options.input)
	root.SetOut(options.output)
	root.SetErr(options.errOutput)
	root.PersistentFlags().StringVar(&options.kubeconfig, "kubeconfig", "", "path to a kubeconfig file")
	root.ValidArgsFunction = completeTargets(options)

	root.AddCommand(newHistoryCommand(options))
	root.AddCommand(newLastCommand(options))
	root.AddCommand(newClearHistoryCommand(options))
	root.AddCommand(newCompletionCommand(root))
	root.AddCommand(newUpdateCommand(options))
	root.AddCommand(newVersionCommand())
	return root
}

func newHistoryCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Select a previous connection",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			entries, err := options.store.List()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				color.New(color.FgYellow).Fprintln(command.OutOrStdout(), "No history yet.")
				return nil
			}
			items := make([]ui.Item, 0, len(entries))
			for _, entry := range entries {
				items = append(items, ui.Item{Name: entry})
			}
			selected, err := ui.Select("Connection history", items, command.InOrStdin(), command.OutOrStdout())
			if errors.Is(err, ui.ErrCancelled) {
				return nil
			}
			if err != nil {
				return err
			}
			return runConnection(command.Context(), options, selected.Name, []string{"sh"})
		},
	}
}

func newLastCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:   "last",
		Short: "Reconnect to the most recent pod",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runLast(command.Context(), options, "1")
		},
	}
}

func runLast(ctx context.Context, options *options, value string) error {
	position, err := strconv.Atoi(value)
	if err != nil || position < 1 {
		return fmt.Errorf("invalid recent connection position %q", value)
	}
	entry, err := options.store.At(position)
	if err != nil {
		return err
	}
	return runConnection(ctx, options, entry, []string{"sh"})
}

func newClearHistoryCommand(options *options) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "clear-history",
		Short: "Clear saved connection history",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if !force {
				fmt.Fprint(command.ErrOrStderr(), "Clear connection history? [y/N] ")
				scanner := bufio.NewScanner(command.InOrStdin())
				if !scanner.Scan() {
					if err := scanner.Err(); err != nil {
						return fmt.Errorf("read confirmation: %w", err)
					}
					return nil
				}
				answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if answer != "y" && answer != "yes" {
					return nil
				}
			}
			if err := options.store.Clear(); err != nil {
				return err
			}
			color.New(color.FgGreen).Fprintln(command.OutOrStdout(), "History cleared.")
			return nil
		},
	}
	command.Flags().BoolVarP(&force, "force", "f", false, "clear history without confirmation")
	return command
}

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	completion := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(command *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(command.OutOrStdout())
			case "zsh":
				return root.GenZshCompletion(command.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(command.OutOrStdout(), true)
			case "powershell":
				return root.GenPowerShellCompletion(command.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
	return completion
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the pod-ssh version",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			fmt.Fprintln(command.OutOrStdout(), Version)
			return nil
		},
	}
}

func newUpdateCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update pod-ssh to the latest release",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			var result updater.Result
			err := ui.Spin(command.Context(), "Checking for updates...", command.InOrStdin(), command.OutOrStdout(), func() error {
				var updateErr error
				result, updateErr = updater.InstallLatest(command.Context(), Version)
				return updateErr
			})
			if err != nil {
				return err
			}
			if !result.Changed {
				color.New(color.FgGreen).Fprintf(command.OutOrStdout(), "Already up to date (%s).\n", result.Current)
				return nil
			}
			color.New(color.FgGreen, color.Bold).Fprintf(
				command.OutOrStdout(),
				"Updated pod-ssh from %s to %s.\n",
				result.Previous,
				result.Current,
			)
			return nil
		},
	}
}

func remoteCommand(args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, true
	}
	if len(args) == 1 && slices.Contains(kube.Shells, args[0]) {
		return args, true
	}
	return args, false
}

func runConnection(ctx context.Context, options *options, targetValue string, args []string) error {
	client, err := kube.New(options.kubeconfig)
	if err != nil {
		return err
	}
	namespace := client.Namespace()
	podQuery := ""
	if targetValue == "" {
		selectedNamespace, err := selectNamespace(ctx, client, options)
		if errors.Is(err, ui.ErrCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		namespace = selectedNamespace
	} else {
		parsed, err := target.Parse(targetValue)
		if err != nil {
			return err
		}
		podQuery = parsed.Pod
		if parsed.Namespace != "" {
			namespace = parsed.Namespace
		}
	}

	var pods []kube.Pod
	if err := ui.Spin(ctx, fmt.Sprintf("Loading pods from %s/%s...", client.Context(), namespace), options.input, options.errOutput, func() error {
		var listErr error
		pods, listErr = client.Pods(ctx, namespace)
		return listErr
	}); err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found in namespace %s", namespace)
	}

	candidates := pods
	if podQuery != "" {
		candidates = kube.MatchingPods(pods, podQuery)
		if len(candidates) == 0 {
			return fmt.Errorf("no pod matching %q in namespace %s", podQuery, namespace)
		}
	}
	podName := candidates[0].Name
	if len(candidates) > 1 || podQuery == "" {
		selected, err := selectPod(namespace, candidates, options)
		if errors.Is(err, ui.ErrCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		podName = selected
	}

	containers, err := client.Containers(ctx, namespace, podName)
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		return fmt.Errorf("pod %s@%s has no containers", podName, namespace)
	}
	container := containers[0]
	if len(containers) > 1 {
		items := make([]ui.Item, 0, len(containers))
		for _, name := range containers {
			items = append(items, ui.Item{Name: name})
		}
		selected, err := ui.Select("Select a container", items, options.input, options.errOutput)
		if errors.Is(err, ui.ErrCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		container = selected.Name
	}

	statusOutput := io.Discard
	if isTerminal(options.errOutput) {
		statusOutput = options.errOutput
	}
	status := newConnectionStatus(statusOutput, fmt.Sprintf("%s@%s (%s)", podName, namespace, container))
	exec := kube.ExecOptions{
		Namespace: namespace,
		Pod:       podName,
		Container: container,
		Stdin:     options.input,
		Stdout:    status.wrap(options.output),
		Stderr:    status.wrap(options.errOutput),
		TTY:       isTerminal(options.input) && isTerminal(options.output),
	}
	command, interactive := remoteCommand(args)
	if interactive {
		preferred := ""
		if len(command) == 1 {
			preferred = command[0]
		}
		err = execShell(ctx, client, exec, status, preferred)
	} else {
		exec.Command = command
		err = client.Exec(ctx, exec)
	}
	if err != nil {
		status.failed()
		return err
	}
	return options.store.Save(target.Target{Pod: podName, Namespace: namespace}.String())
}

func isTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func execShell(ctx context.Context, client *kube.Client, exec kube.ExecOptions, status *connectionStatus, preferred string) error {
	if !exec.TTY {
		return fmt.Errorf("an interactive shell requires a terminal")
	}
	shell, err := client.FindShell(ctx, exec.Namespace, exec.Pod, exec.Container, append([]string{preferred}, kube.Shells...)...)
	if err != nil {
		return err
	}
	if preferred != "" && shell != preferred {
		status.note(fmt.Sprintf("Shell %q is unavailable; using %q.", preferred, shell))
	}
	exec.Command = []string{shell}
	return client.Exec(ctx, exec)
}

func selectNamespace(ctx context.Context, client *kube.Client, options *options) (string, error) {
	var namespaces []string
	if err := ui.Spin(ctx, "Loading namespaces from "+client.Context()+"...", options.input, options.errOutput, func() error {
		var listErr error
		namespaces, listErr = client.Namespaces(ctx)
		return listErr
	}); err != nil {
		return "", err
	}
	if len(namespaces) == 0 {
		return "", fmt.Errorf("no namespaces found in context %s", client.Context())
	}
	items := make([]ui.Item, 0, len(namespaces))
	for _, namespace := range namespaces {
		detail := ""
		if namespace == client.Namespace() {
			detail = "current"
		}
		items = append(items, ui.Item{Name: namespace, Detail: detail})
	}
	selected, err := ui.Select("Namespaces in "+client.Context(), items, options.input, options.errOutput)
	if err != nil {
		return "", err
	}
	return selected.Name, nil
}

func selectPod(namespace string, pods []kube.Pod, options *options) (string, error) {
	items := make([]ui.Item, 0, len(pods))
	for _, pod := range pods {
		items = append(items, ui.Item{
			Name: pod.Name,
			Detail: fmt.Sprintf(
				"%d/%d ready · %s · %d restarts · %s",
				pod.Ready,
				pod.Containers,
				pod.Phase,
				pod.Restarts,
				shortDuration(pod.Age),
			),
		})
	}
	selected, err := ui.Select("Pods in "+namespace, items, options.input, options.errOutput)
	if err != nil {
		return "", err
	}
	return selected.Name, nil
}

func shortDuration(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

func completeTargets(options *options) cobra.CompletionFunc {
	return func(command *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx, cancel := context.WithTimeout(command.Context(), 2*time.Second)
		defer cancel()
		client, err := kube.New(options.kubeconfig)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) > 1 || (len(args) == 1 && strings.Contains(toComplete, "/")) {
			return completeRemotePaths(ctx, client, args[0], toComplete)
		}
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if before, after, ok := strings.Cut(toComplete, "@"); ok {
			podQuery := before
			namespaceQuery := after
			namespaces, listErr := client.Namespaces(ctx)
			if listErr != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			matches := make([]string, 0, len(namespaces))
			for _, namespace := range namespaces {
				if strings.Contains(namespace, namespaceQuery) {
					matches = append(matches, podQuery+"@"+namespace+"\tnamespace")
				}
			}
			return matches, cobra.ShellCompDirectiveNoFileComp
		}

		namespace := client.Namespace()
		pods, err := client.Pods(ctx, namespace)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		matches := make([]string, 0, len(pods))
		for _, pod := range pods {
			candidate := pod.Name + "@" + namespace
			if strings.Contains(candidate, toComplete) {
				matches = append(matches, candidate+"\t"+pod.Phase)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

const remoteGlob = `for f in "$1"*; do if [ -d "$f" ]; then echo "$f/"; elif [ -e "$f" ]; then echo "$f"; fi; done`

func completeRemotePaths(ctx context.Context, client *kube.Client, targetValue, toComplete string) ([]string, cobra.ShellCompDirective) {
	parsed, err := target.Parse(targetValue)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	namespace := client.Namespace()
	if parsed.Namespace != "" {
		namespace = parsed.Namespace
	}
	pods, err := client.Pods(ctx, namespace)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	matches := kube.MatchingPods(pods, parsed.Pod)
	if len(matches) != 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	containers, err := client.Containers(ctx, namespace, matches[0].Name)
	if err != nil || len(containers) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	shell, err := client.FindShell(ctx, namespace, matches[0].Name, containers[0], kube.Shells...)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var output bytes.Buffer
	if err := client.Exec(ctx, kube.ExecOptions{
		Namespace: namespace,
		Pod:       matches[0].Name,
		Container: containers[0],
		Command:   []string{shell, "-c", remoteGlob, shell, toComplete},
		Stdout:    &output,
		Stderr:    io.Discard,
	}); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return pathCompletions(output.String())
}

func pathCompletions(output string) ([]string, cobra.ShellCompDirective) {
	paths := []string{}
	for line := range strings.Lines(output) {
		if path := strings.TrimSuffix(line, "\n"); path != "" {
			paths = append(paths, path)
		}
	}
	directive := cobra.ShellCompDirectiveNoFileComp
	if len(paths) == 1 && strings.HasSuffix(paths[0], "/") {
		directive |= cobra.ShellCompDirectiveNoSpace
	}
	return paths, directive
}
