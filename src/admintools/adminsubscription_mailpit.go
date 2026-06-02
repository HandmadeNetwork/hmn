package admintools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
)

type membershipMailpitContextKey struct{}

func withMembershipMailpit(ctx context.Context, m *membershipMailpit) context.Context {
	return context.WithValue(ctx, membershipMailpitContextKey{}, m)
}

func membershipMailpitFromContext(ctx context.Context) *membershipMailpit {
	if ctx == nil {
		return nil
	}
	if m, ok := ctx.Value(membershipMailpitContextKey{}).(*membershipMailpit); ok {
		return m
	}
	return nil
}

type membershipMailpit struct {
	httpBaseURL string
	smtpAddr    string
	cmd         *exec.Cmd
	client      *http.Client
}

func startMembershipMailpit() (*membershipMailpit, bool, error) {
	_, err := exec.LookPath("mailpit")
	if err != nil {
		return nil, false, nil
	}

	smtpPort, err := reserveTCPPort()
	if err != nil {
		return nil, true, fmt.Errorf("reserve smtp port: %w", err)
	}
	httpPort, err := reserveTCPPort()
	if err != nil {
		return nil, true, fmt.Errorf("reserve http port: %w", err)
	}

	m := &membershipMailpit{
		httpBaseURL: fmt.Sprintf("http://127.0.0.1:%d", httpPort),
		smtpAddr:    fmt.Sprintf("127.0.0.1:%d", smtpPort),
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
	m.cmd = exec.Command("mailpit",
		"--smtp", fmt.Sprintf("127.0.0.1:%d", smtpPort),
		"--listen", fmt.Sprintf("127.0.0.1:%d", httpPort),
	)
	if err := m.cmd.Start(); err != nil {
		return nil, true, fmt.Errorf("start mailpit: %w", err)
	}

	if err := m.waitReady(10 * time.Second); err != nil {
		_ = m.Stop()
		return nil, true, err
	}

	config.Config.Email.ServerAddress = "127.0.0.1"
	config.Config.Email.ServerPort = smtpPort
	config.Config.Email.MailerUsername = ""
	config.Config.Email.MailerPassword = ""
	config.Config.Email.ForceToAddress = ""

	if err := m.ClearMessages(); err != nil {
		_ = m.Stop()
		return nil, true, err
	}

	return m, true, nil
}

func (m *membershipMailpit) Stop() error {
	if m == nil || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	_ = m.cmd.Process.Kill()
	_, _ = m.cmd.Process.Wait()
	return nil
}

func (m *membershipMailpit) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, m.httpBaseURL+"/api/v1/info", nil)
		if err == nil {
			resp, err := m.client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("mailpit did not become ready at %s", m.httpBaseURL)
}

func (m *membershipMailpit) ClearMessages() error {
	req, err := http.NewRequest(http.MethodDelete, m.httpBaseURL+"/api/v1/messages", nil)
	if err != nil {
		return err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mailpit clear messages returned %d", resp.StatusCode)
	}
	return nil
}

func (m *membershipMailpit) messageSubjects() ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, m.httpBaseURL+"/api/v1/messages?start=0&limit=200", nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mailpit list messages returned %d", resp.StatusCode)
	}

	var payload struct {
		Messages []struct {
			Subject string `json:"Subject"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	// Mailpit returns newest-first; reverse so assertions read chronologically.
	subjects := make([]string, 0, len(payload.Messages))
	for i := len(payload.Messages) - 1; i >= 0; i-- {
		subjects = append(subjects, payload.Messages[i].Subject)
	}
	return subjects, nil
}

func (m *membershipMailpit) WaitForSubjects(expected []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		subjects, err := m.messageSubjects()
		if err == nil {
			if err := assertSubjectsEqual(subjects, expected); err == nil {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	subjects, err := m.messageSubjects()
	if err != nil {
		return err
	}
	return assertSubjectsEqual(subjects, expected)
}

func assertSubjectsEqual(actual, expected []string) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("email subject count mismatch: got %d expected %d (actual: %v)", len(actual), len(expected), actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			return fmt.Errorf("email subject mismatch at index %d: got %q expected %q (actual: %v)", i, actual[i], expected[i], actual)
		}
	}
	return nil
}

func reserveTCPPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("listener is not TCP")
	}
	return addr.Port, nil
}
