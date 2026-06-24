// Package docker is a Docker driver for togo deploy: it builds the app image,
// (optionally) pushes it to a registry, and runs it on the target host over SSH
// (or locally when no host is set). Select with DEPLOY_PROVIDER=docker.
package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/togo-framework/deploy"
	"github.com/togo-framework/togo"
)

func init() {
	deploy.RegisterDriver("docker", func(k *togo.Kernel) (deploy.Deployer, error) {
		return &driver{
			registry: os.Getenv("DOCKER_REGISTRY"),
			k:        k,
		}, nil
	})
}

type driver struct {
	registry string
	k        *togo.Kernel
}

func (d *driver) image(spec deploy.Spec) string {
	if spec.Image != "" {
		return spec.Image
	}
	name := spec.App
	if name == "" {
		name = "app"
	}
	if d.registry != "" {
		return strings.TrimRight(d.registry, "/") + "/" + name + ":latest"
	}
	return name + ":latest"
}

// run executes a command, returning combined output.
func run(ctx context.Context, dir, name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", fmt.Errorf("deploy-docker: %q not found on PATH", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

func (d *driver) Provision(ctx context.Context, spec deploy.Spec) (*deploy.Result, error) {
	// Nothing to provision for a plain Docker target beyond ensuring docker exists.
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("deploy-docker: docker not installed")
	}
	return &deploy.Result{Message: "docker ready"}, nil
}

func (d *driver) Deploy(ctx context.Context, spec deploy.Spec) (*deploy.Result, error) {
	img := d.image(spec)
	dir := spec.Dir
	if dir == "" {
		dir = "."
	}
	// Build the image.
	if out, err := run(ctx, dir, "docker", "build", "-t", img, "."); err != nil {
		return nil, fmt.Errorf("docker build: %w\n%s", err, out)
	}
	// Push when a registry is configured.
	if d.registry != "" {
		if out, err := run(ctx, dir, "docker", "push", img); err != nil {
			return nil, fmt.Errorf("docker push: %w\n%s", err, out)
		}
	}
	// Run: on the host over SSH if Host is set, else locally.
	runArgs := []string{"run", "-d", "--name", spec.App, "--restart", "unless-stopped"}
	for k, v := range spec.Env {
		runArgs = append(runArgs, "-e", k+"="+v)
	}
	runArgs = append(runArgs, "-p", "8080:8080", img)
	if spec.Host != "" {
		ssh := append([]string{sshTarget(spec), "docker", "rm", "-f", spec.App, ";", "docker"}, append([]string{"pull", img, ";", "docker"}, runArgs...)...)
		if out, err := run(ctx, dir, "ssh", ssh...); err != nil {
			return nil, fmt.Errorf("remote docker run: %w\n%s", err, out)
		}
		return &deploy.Result{Message: "deployed " + img + " on " + spec.Host, URL: urlFor(spec)}, nil
	}
	_, _ = run(ctx, dir, "docker", "rm", "-f", spec.App)
	if out, err := run(ctx, dir, "docker", runArgs...); err != nil {
		return nil, fmt.Errorf("docker run: %w\n%s", err, out)
	}
	return &deploy.Result{Message: "running " + img + " locally", URL: "http://localhost:8080"}, nil
}

func (d *driver) Destroy(ctx context.Context, spec deploy.Spec) error {
	if spec.Host != "" {
		_, err := run(ctx, "", "ssh", sshTarget(spec), "docker", "rm", "-f", spec.App)
		return err
	}
	_, err := run(ctx, "", "docker", "rm", "-f", spec.App)
	return err
}

func (d *driver) Status(ctx context.Context, spec deploy.Spec) (*deploy.Status, error) {
	args := []string{"inspect", "-f", "{{.State.Running}}", spec.App}
	var out string
	var err error
	if spec.Host != "" {
		out, err = run(ctx, "", "ssh", append([]string{sshTarget(spec), "docker"}, args...)...)
	} else {
		out, err = run(ctx, "", "docker", args...)
	}
	if err != nil {
		return &deploy.Status{Healthy: false, Detail: strings.TrimSpace(out)}, nil
	}
	healthy := strings.TrimSpace(out) == "true"
	return &deploy.Status{Healthy: healthy, Detail: strings.TrimSpace(out)}, nil
}

func sshTarget(spec deploy.Spec) string {
	user := spec.User
	if user == "" {
		user = "root"
	}
	return user + "@" + spec.Host
}

func urlFor(spec deploy.Spec) string {
	if spec.Domain != "" {
		return "https://" + spec.Domain
	}
	return "http://" + spec.Host + ":8080"
}
