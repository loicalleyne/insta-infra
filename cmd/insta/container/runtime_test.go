package container

import (
	"os"
	"os/exec"
	"testing"
)

// MockRuntime implements Runtime interface for testing
type MockRuntime struct {
	available         bool
	name              string
	composeUpCalled   bool
	composeDownCalled bool
	execCalled        bool
	lastServices      []string
	lastCmd           string
}

func NewMockRuntime(name string, available bool) *MockRuntime {
	return &MockRuntime{
		name:      name,
		available: available,
	}
}

func (m *MockRuntime) Name() string {
	return m.name
}

func (m *MockRuntime) CheckAvailable() error {
	if !m.available {
		return &exec.Error{Name: m.name, Err: exec.ErrNotFound}
	}
	return nil
}

func (m *MockRuntime) ComposeUp(composeFiles []string, services []string, quiet bool) error {
	m.composeUpCalled = true
	m.lastServices = services
	return nil
}

func (m *MockRuntime) ComposeDown(composeFiles []string, services []string) error {
	m.composeDownCalled = true
	m.lastServices = services
	return nil
}

func (m *MockRuntime) ExecInContainer(containerName string, cmd string, interactive bool) error {
	m.execCalled = true
	m.lastCmd = cmd
	return nil
}

func TestNewProvider(t *testing.T) {
	provider := NewProvider()
	if provider == nil {
		t.Fatal("NewProvider returned nil")
	}
	if len(provider.runtimes) != 2 {
		t.Fatalf("Expected 2 runtimes, got %d", len(provider.runtimes))
	}
}

func TestProviderDetectRuntime(t *testing.T) {
	// Case 1: No runtimes available
	p1 := &Provider{
		runtimes: []Runtime{
			NewMockRuntime("docker", false),
			NewMockRuntime("podman", false),
		},
	}

	if err := p1.DetectRuntime(); err == nil {
		t.Fatal("Expected error when no runtimes are available")
	}

	// Case 2: Docker available
	p2 := &Provider{
		runtimes: []Runtime{
			NewMockRuntime("docker", true),
			NewMockRuntime("podman", false),
		},
	}

	if err := p2.DetectRuntime(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if p2.selected == nil {
		t.Fatal("No runtime selected")
	}

	if p2.selected.Name() != "docker" {
		t.Fatalf("Expected docker runtime, got %s", p2.selected.Name())
	}

	// Case 3: Podman available
	p3 := &Provider{
		runtimes: []Runtime{
			NewMockRuntime("docker", false),
			NewMockRuntime("podman", true),
		},
	}

	if err := p3.DetectRuntime(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if p3.selected == nil {
		t.Fatal("No runtime selected")
	}

	if p3.selected.Name() != "podman" {
		t.Fatalf("Expected podman runtime, got %s", p3.selected.Name())
	}
}

func TestDockerCheckAvailable(t *testing.T) {
	// Skip if docker is not installed
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}

	docker := NewDockerRuntime()
	err = docker.CheckAvailable()
	if err != nil {
		// Skip if Docker daemon is not running
		t.Skipf("Docker daemon not running, skipping test: %v", err)
	}
}

func TestDockerComposeOperations(t *testing.T) {
	// Use mock runtime for compose operations
	mock := NewMockRuntime("docker", true)

	// Test ComposeUp
	err := mock.ComposeUp([]string{"docker-compose.yaml"}, []string{"service1"}, false)
	if err != nil {
		t.Fatalf("Unexpected error in ComposeUp: %v", err)
	}
	if !mock.composeUpCalled {
		t.Fatal("ComposeUp was not called")
	}
	if len(mock.lastServices) != 1 || mock.lastServices[0] != "service1" {
		t.Fatalf("Expected service1, got %v", mock.lastServices)
	}

	// Test ComposeDown
	err = mock.ComposeDown([]string{"docker-compose.yaml"}, []string{"service1"})
	if err != nil {
		t.Fatalf("Unexpected error in ComposeDown: %v", err)
	}
	if !mock.composeDownCalled {
		t.Fatal("ComposeDown was not called")
	}
}

func TestExecInContainer(t *testing.T) {
	// Use mock runtime for exec operation
	mock := NewMockRuntime("docker", true)

	// Test ExecInContainer
	err := mock.ExecInContainer("container1", "echo hello", false)
	if err != nil {
		t.Fatalf("Unexpected error in ExecInContainer: %v", err)
	}
	if !mock.execCalled {
		t.Fatal("ExecInContainer was not called")
	}
	if mock.lastCmd != "echo hello" {
		t.Fatalf("Expected 'echo hello', got '%s'", mock.lastCmd)
	}
}

func TestPodmanCheckAvailable(t *testing.T) {
	// Skip if podman is not installed
	_, err := exec.LookPath("podman")
	if err != nil {
		t.Skip("Podman not available, skipping test")
	}

	podman := NewPodmanRuntime()
	err = podman.CheckAvailable()
	if err != nil {
		// Skip if Podman is not properly configured
		t.Skipf("Podman not properly configured, skipping test: %v", err)
	}
}

func TestComposeFilesHandling(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "runtime-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test mock runtime
	t.Run("mock runtime compose handling", func(t *testing.T) {
		mockRuntime := NewMockRuntime("docker", true)
		composeFiles := []string{"docker-compose.yaml", "docker-compose-persist.yaml"}
		services := []string{"postgres", "mysql"}

		err := mockRuntime.ComposeUp(composeFiles, services, true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !mockRuntime.composeUpCalled {
			t.Error("Expected ComposeUp to be called")
		}

		if len(mockRuntime.lastServices) != len(services) {
			t.Errorf("Expected %d services, got %d", len(services), len(mockRuntime.lastServices))
		}
	})
}
