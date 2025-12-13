package kiro

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"budgie/internal/health"
	"github.com/google/uuid"
)

type Executor struct {
	binary  string
	timeout time.Duration
	monitor *health.Monitor
}

type Result struct {
	Output    string
	SessionID string
	Error     error
	Duration  time.Duration
	Retried   bool
}

func NewExecutor(binary string, timeout time.Duration, monitor *health.Monitor) *Executor {
	return &Executor{
		binary:  binary,
		timeout: timeout,
		monitor: monitor,
	}
}

// GetUniqueResponseFile generates a unique response filename for a workspace
func GetUniqueResponseFile(sessionDir string) string {
	for {
		responseFile := fmt.Sprintf("response-%s.txt", uuid.New().String()[:8])
		responsePath := filepath.Join(sessionDir, responseFile)
		if _, err := os.Stat(responsePath); os.IsNotExist(err) {
			return responseFile
		}
	}
}

func (e *Executor) Execute(ctx context.Context, agentName, prompt, sessionDir, sessionID, model string) Result {
	start := time.Now()

	result := e.executeOnce(ctx, agentName, prompt, sessionDir, sessionID, model)

	if result.Error != nil && shouldRetry(result.Error) {
		time.Sleep(2 * time.Second)
		retryResult := e.executeOnce(ctx, agentName, prompt, sessionDir, sessionID, model)
		retryResult.Retried = true

		if retryResult.Error == nil {
			if e.monitor != nil {
				e.monitor.RecordSuccess(agentName, time.Since(start))
			}
			return retryResult
		}

		result = retryResult
	}

	if e.monitor != nil {
		duration := time.Since(start)
		if result.Error != nil {
			isTimeout := strings.Contains(result.Error.Error(), "timeout") ||
				strings.Contains(result.Error.Error(), "deadline exceeded")
			e.monitor.RecordFailure(agentName, duration, result.Error.Error(), isTimeout)
		} else {
			e.monitor.RecordSuccess(agentName, duration)
		}
	}

	return result
}

func (e *Executor) executeOnce(ctx context.Context, agentName, prompt, sessionDir, sessionID, model string) Result {
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	args := []string{"chat", "--agent", agentName, "--no-interactive"}

	if model != "" {
		args = append(args, "--model", model)
	}

	if sessionID != "" {
		args = append(args, "--resume")
	}

	args = append(args, prompt)

	cmd := exec.CommandContext(timeoutCtx, e.binary, args...)
	cmd.Dir = sessionDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return Result{Error: fmt.Errorf("agent timeout after %v", e.timeout)}
		}

		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return Result{Error: fmt.Errorf("kiro-cli failed: %s", errMsg)}
	}

	return Result{
		Output: strings.TrimSpace(stdout.String()),
		Error:  nil,
	}
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return true
	}

	if strings.Contains(errStr, "signal") || strings.Contains(errStr, "killed") {
		return true
	}

	if strings.Contains(errStr, "exit status") && !strings.Contains(errStr, "required") {
		return true
	}

	return false
}
