package dev

import (
	"io"
	"testing"
)

type TestCommander struct {
	Path   string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Writer
}

func (tc *TestCommander) Run() error {
	return nil
}

func (tc *TestCommander) NewCommand(name string, args ...string) Command {
	tc.Path = name

	for _, arg := range args {
		tc.Args = append(tc.Args, arg)
	}

	return tc
}

func NewTestCommander() *TestCommander {
	return &TestCommander{}
}

var tc TestCommander

func setup() {
	tc = TestCommander{}
}

func TestRunComposeBuild(t *testing.T) {
	tests := []struct {
		Project      string
		ComposePaths []string
		Args         []string
		Expected     []string
	}{
		{"foo", []string{"/foo/bar/baz"}, []string{}, []string{"-p", "foo", "-f", "/foo/bar/baz", "build"}},
		{"far", []string{"/foo/bar/baz", "/boo/far/faz"}, []string{"--no-skippy"}, []string{"-p", "far", "-f", "/foo/bar/baz", "-f", "/boo/far/faz", "build", "--no-skippy"}},
	}

	for _, test := range tests {
		setup()
		setExecutor(tc.NewCommand)

		RunComposeBuild(test.Project, test.ComposePaths, test.Args...)

		if tc.Path != "docker-compose" {
			t.Errorf("Expected path be %s but got %s", "docker-compose", tc.Path)
		}

		expectedArguments := len(test.Expected)
		if expectedArguments != len(tc.Args) {
			t.Errorf("Expected %d arguments but received %d", expectedArguments, len(tc.Args))
		}

		for i, arg := range tc.Args {
			if arg != test.Expected[i] {
				t.Errorf("Expected argument %d to be '%s' but got '%s'", i, test.Expected[i], tc.Args[i])
			}
		}

	}
}

func TestRunComposeUp(t *testing.T) {
	tests := []struct {
		Project      string
		ComposePaths []string
		Args         []string
		Expected     []string
	}{
		{"foo", []string{"/foo/bar/baz"}, []string{}, []string{"-p", "foo", "-f", "/foo/bar/baz", "up"}},
		{"far", []string{"/foo/bar/baz", "/boo/far/faz"}, []string{"--no-skippy"}, []string{"-p", "far", "-f", "/foo/bar/baz", "-f", "/boo/far/faz", "up", "--no-skippy"}},
	}

	for _, test := range tests {
		setup()
		setExecutor(tc.NewCommand)

		RunComposeUp(test.Project, test.ComposePaths, test.Args...)

		if tc.Path != "docker-compose" {
			t.Errorf("Expected path be %s but got %s", "docker-compose", tc.Path)
		}

		expectedArguments := len(test.Expected)
		if expectedArguments != len(tc.Args) {
			t.Errorf("Expected %d arguments but received %d", expectedArguments, len(tc.Args))
		}

		for i, arg := range tc.Args {
			if arg != test.Expected[i] {
				t.Errorf("Expected argument %d to be '%s' but got '%s'", i, test.Expected[i], tc.Args[i])
			}
		}
	}
}

func TestRunComposePs(t *testing.T) {
	tests := []struct {
		Project      string
		ComposePaths []string
		Args         []string
		Expected     []string
	}{
		{"foo", []string{"/foo/bar/baz"}, []string{}, []string{"-p", "foo", "-f", "/foo/bar/baz", "ps"}},
		{"far", []string{"/foo/bar/baz", "/boo/far/faz"}, []string{"--no-skippy"}, []string{"-p", "far", "-f", "/foo/bar/baz", "-f", "/boo/far/faz", "ps", "--no-skippy"}},
	}

	for _, test := range tests {
		setup()
		setExecutor(tc.NewCommand)

		RunComposePs(test.Project, test.ComposePaths, test.Args...)

		if tc.Path != "docker-compose" {
			t.Errorf("Expected path be %s but got %s", "docker-compose", tc.Path)
		}

		expectedArguments := len(test.Expected)
		if expectedArguments != len(tc.Args) {
			t.Errorf("Expected %d arguments but received %d", expectedArguments, len(tc.Args))
		}

		for i, arg := range tc.Args {
			if arg != test.Expected[i] {
				t.Errorf("Expected argument %d to be '%s' but got '%s'", i, test.Expected[i], tc.Args[i])
			}
		}
	}
}

func TestRunComposeLogs(t *testing.T) {
	tests := []struct {
		Project      string
		ComposePaths []string
		Args         []string
		Expected     []string
	}{
		{"foo", []string{"/foo/bar/baz"}, []string{}, []string{"-p", "foo", "-f", "/foo/bar/baz", "logs"}},
		{"far", []string{"/foo/bar/baz", "/boo/far/faz"}, []string{"--no-skippy"}, []string{"-p", "far", "-f", "/foo/bar/baz", "-f", "/boo/far/faz", "logs", "--no-skippy"}},
	}

	for _, test := range tests {
		setup()
		setExecutor(tc.NewCommand)

		RunComposeLogs(test.Project, test.ComposePaths, test.Args...)

		if tc.Path != "docker-compose" {
			t.Errorf("Expected path be %s but got %s", "docker-compose", tc.Path)
		}

		expectedArguments := len(test.Expected)
		if expectedArguments != len(tc.Args) {
			t.Errorf("Expected %d arguments but received %d", expectedArguments, len(tc.Args))
			return
		}

		for i, arg := range tc.Args {
			if arg != test.Expected[i] {
				t.Errorf("Expected argument %d to be '%s' but got '%s'", i, test.Expected[i], tc.Args[i])
			}
		}
	}
}

func TestRunComposeDown(t *testing.T) {
	tests := []struct {
		Project      string
		ComposePaths []string
		Args         []string
		Expected     []string
	}{
		{"foo", []string{"/foo/bar/baz"}, []string{}, []string{"-p", "foo", "-f", "/foo/bar/baz", "down"}},
		{"far", []string{"/foo/bar/baz", "/boo/far/faz"}, []string{"--no-skippy"}, []string{"-p", "far", "-f", "/foo/bar/baz", "-f", "/boo/far/faz", "down", "--no-skippy"}},
	}

	for _, test := range tests {
		setup()
		setExecutor(tc.NewCommand)

		RunComposeDown(test.Project, test.ComposePaths, test.Args...)

		if tc.Path != "docker-compose" {
			t.Errorf("Expected path be %s but got %s", "docker-compose", tc.Path)
		}

		expectedArguments := len(test.Expected)
		if expectedArguments != len(tc.Args) {
			t.Errorf("Expected %d arguments but received %d", expectedArguments, len(tc.Args))
			return
		}

		for i, arg := range tc.Args {
			if arg != test.Expected[i] {
				t.Errorf("Expected argument %d to be '%s' but got '%s'", i, test.Expected[i], tc.Args[i])
			}
		}
	}
}

func TestRunOnContainer(t *testing.T) {
	tests := []struct {
		ContainerName string
		Args          []string
		Expected      []string
	}{
		{"foo", []string{"ls", "-al"}, []string{"exec", "foo", "ls", "-al"}},
	}

	for _, test := range tests {
		setup()
		setExecutor(tc.NewCommand)

		RunOnContainer(test.ContainerName, test.Args...)

		if tc.Path != "docker" {
			t.Errorf("Expected path be %s but got %s", "docker", tc.Path)
		}

		expectedArguments := len(test.Expected)
		if expectedArguments != len(tc.Args) {
			t.Errorf("Expected %d arguments but received %d", expectedArguments, len(tc.Args))
		}

		for i, arg := range tc.Args {
			if arg != test.Expected[i] {
				t.Errorf("Expected argument %d to be '%s' but got '%s'", i, test.Expected[i], tc.Args[i])
			}
		}
	}
}
