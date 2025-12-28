package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/pardnchiu/go-faas/internal/container"
)

type SSE struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Type  string      `json:"type,omitempty"`
}

func sendEvent(w http.ResponseWriter, flusher http.Flusher, event, output string) {
	stream := SSE{Event: event}

	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		switch v := data.(type) {
		case string:
			stream.Data = v
			stream.Type = "string"
		case float64, int, int64, json.Number:
			stream.Data = v
			stream.Type = "number"
		default:
			stream.Data = v
			stream.Type = "json"
		}
	} else {
		stream.Data = output
		stream.Type = "text"
	}

	b, _ := json.Marshal(stream)
	fmt.Fprintf(w, "data: %s\n\n", b)
	flusher.Flush()
}

func sendDone(w http.ResponseWriter, flusher http.Flusher, event, msg string) {
	sendEvent(w, flusher, event, msg)
	flusher.Flush()

	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	_ = conn.Close()
}

func runScriptWithSSE(code, lang, input string, w http.ResponseWriter, flusher http.Flusher, clientCtx context.Context) (string, error) {
	ct, runtime, localPath, wrapPath, ctPath, err := prepareScript(code, lang)
	defer func() {
		container.Release(ct)
		if err := os.Remove(localPath); err != nil {
			slog.Warn("failed to cleanup temp file",
				slog.String("file", localPath),
				slog.String("error", err.Error()),
			)
		}
	}()
	if err != nil {
		return "", err
	}

	ctx, execCancel := context.WithTimeout(context.Background(), timeoutRequest)
	defer execCancel()

	var cmd *exec.Cmd
	if lang == "python" {
		cmd = exec.CommandContext(ctx, "podman", "exec", "-i", ct, runtime, "-u", wrapPath, ctPath)
	} else {
		cmd = exec.CommandContext(ctx, "podman", "exec", "-i", ct, runtime, wrapPath, ctPath)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		io.WriteString(stdin, input)
		stdin.Close()
	}()

	outScanner := bufio.NewScanner(stdout)
	errScanner := bufio.NewScanner(stderr)
	var lastLine string

	doneChan := make(chan struct{}, 2)
	errChan := make(chan string, 1)

	go func() {
		defer func() {
			doneChan <- struct{}{}
		}()

		// * only send previous line as log
		var prev string
		for outScanner.Scan() {
			if prev != "" {
				sendEvent(w, flusher, "log", prev)
				flusher.Flush()
			}
			prev = outScanner.Text()
		}
		lastLine = strings.TrimSpace(prev)
	}()

	go func() {
		defer func() { doneChan <- struct{}{} }()

		for errScanner.Scan() {
			select {
			// * send any error when found, and stop script
			case errChan <- errScanner.Text():
			default:
			}
		}
	}()

	procDone := make(chan error, 1)
	go func() {
		procDone <- cmd.Wait()
	}()

	var resultErr error
	select {
	case <-clientCtx.Done():
		// * request canceled
		_ = cmd.Process.Kill()
		resultErr = fmt.Errorf("stopped to run script: client disconnected")
	case <-ctx.Done():
		// * execution timeout
		_ = cmd.Process.Kill()
		if ctx.Err() == context.DeadlineExceeded {
			resultErr = fmt.Errorf("stopped to run script: timeout (max %v)", timeoutRequest)
		} else {
			resultErr = fmt.Errorf("stopped to run script: canceled")
		}
	case errMsg := <-errChan:
		// * received error output
		_ = cmd.Process.Kill()
		resultErr = fmt.Errorf("stopped to run script: %s", strings.TrimSpace(errMsg))
	case err := <-procDone:
		// * no error output, but exit code != 0
		if err != nil {
			resultErr = fmt.Errorf("stopped to run script: %w", err)
		}
	}

	<-doneChan
	<-doneChan

	if resultErr != nil {
		return "", resultErr
	}
	return strings.TrimSpace(lastLine), nil
}
