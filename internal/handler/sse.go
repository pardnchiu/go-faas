package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pardnchiu/go-faas/internal/sandbox"
	"github.com/pardnchiu/go-faas/internal/utils"
)

type SSE struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
	Type  string `json:"type,omitempty"`
}

func sendEvent(w http.ResponseWriter, flusher http.Flusher, event, output string) {
	stream := SSE{Event: event}

	var data any
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
	if timeoutScript == 0 {
		timeoutScript = time.Duration(utils.GetWithDefaultInt("TIMEOUT_SCRIPT", 30)) * time.Second
		timeoutRequest = timeoutScript + timeoutRedis
	}

	ctx, execCancel := context.WithTimeout(context.Background(), timeoutRequest)
	defer execCancel()

	cmd, err := sandbox.SandboxCommand(ctx, lang)
	if err != nil {
		return "", fmt.Errorf("sandbox command: %w", err)
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

	// * prepare stdin with JSON containing code and input
	go func() {
		payload := map[string]string{
			"code":  code,
			"input": input,
		}
		payloadBody, _ := json.Marshal(payload)
		io.WriteString(stdin, string(payloadBody))
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
