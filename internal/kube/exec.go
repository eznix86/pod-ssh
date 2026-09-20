package kube

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecOptions configures an interactive command in a container.
type ExecOptions struct {
	Namespace string
	Pod       string
	Container string
	Command   []string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

// Exec starts an interactive TTY command in a pod container.
func (c *Client) Exec(ctx context.Context, options ExecOptions) error {
	request := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.Pod).
		Namespace(options.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Command:   options.Command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	websocketExecutor, err := remotecommand.NewWebSocketExecutor(c.restConfig, http.MethodGet, request.URL().String())
	if err != nil {
		return fmt.Errorf("create WebSocket executor: %w", err)
	}
	spdyExecutor, err := remotecommand.NewSPDYExecutor(c.restConfig, http.MethodPost, request.URL())
	if err != nil {
		return fmt.Errorf("create SPDY executor: %w", err)
	}
	executor, err := remotecommand.NewFallbackExecutor(
		websocketExecutor,
		spdyExecutor,
		func(streamErr error) bool {
			return httpstream.IsUpgradeFailure(streamErr) || httpstream.IsHTTPSProxyError(streamErr)
		},
	)
	if err != nil {
		return fmt.Errorf("create fallback executor: %w", err)
	}

	file, ok := options.Stdin.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return fmt.Errorf("interactive exec requires a terminal")
	}
	state, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return fmt.Errorf("enable raw terminal mode: %w", err)
	}
	defer func() { _ = term.Restore(int(file.Fd()), state) }()

	resizeQueue := newSizeQueue(ctx, file)
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             options.Stdin,
		Stdout:            options.Stdout,
		Stderr:            options.Stderr,
		Tty:               true,
		TerminalSizeQueue: resizeQueue,
	}); err != nil {
		return fmt.Errorf("exec %v in %s@%s: %w", options.Command, options.Pod, options.Namespace, err)
	}
	return nil
}

type sizeQueue struct {
	sizes <-chan *remotecommand.TerminalSize
}

func newSizeQueue(ctx context.Context, terminal *os.File) *sizeQueue {
	sizes := make(chan *remotecommand.TerminalSize, 1)
	pushSize := func() {
		width, height, err := term.GetSize(int(terminal.Fd()))
		if err != nil || width < 1 || height < 1 {
			return
		}
		size := &remotecommand.TerminalSize{Width: uint16(width), Height: uint16(height)}
		select {
		case sizes <- size:
		default:
		}
	}
	pushSize()
	go func() {
		defer close(sizes)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pushSize()
			}
		}
	}()
	return &sizeQueue{sizes: sizes}
}

func (q *sizeQueue) Next() *remotecommand.TerminalSize {
	return <-q.sizes
}
