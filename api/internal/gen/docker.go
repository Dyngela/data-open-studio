package gen

import (
	"api"
	"api/pkg"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

//go:embed all:lib
var libFS embed.FS

// knownDriverVersions maps database driver import paths to pinned versions
var knownDriverVersions = map[string]string{
	"github.com/lib/pq":                "v1.10.9",
	"github.com/denisenkom/go-mssqldb": "v0.12.3",
	"github.com/go-sql-driver/mysql":   "v1.8.1",
}

// fixedDependencies are always included in generated go.mod (required by lib/)
var fixedDependencies = map[string]string{
	"github.com/nats-io/nats.go": "v1.48.0",
}

const dockerfileContent = `FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o /job .
CMD ["/job"]
`

// prepareWorkspace creates an isolated directory with all files needed to build the job Docker image
func (j *JobExecution) prepareWorkspace() (string, error) {
	workDir, err := os.MkdirTemp("", "job-*")
	if err != nil {
		return "", fmt.Errorf("failed to create workspace: %w", err)
	}

	// Write generated source code
	if err := j.writeToFile(filepath.Join(workDir, "main.go")); err != nil {
		return workDir, err
	}

	// Copy the lib package into the workspace
	if err := extractLib(workDir); err != nil {
		return workDir, fmt.Errorf("failed to write lib: %w", err)
	}

	// Write go.mod with required database drivers
	goMod := j.generateGoMod()
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return workDir, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Write Dockerfile
	if err := os.WriteFile(filepath.Join(workDir, "Dockerfile"), []byte(dockerfileContent), 0644); err != nil {
		return workDir, fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return workDir, nil
}

// extractLib copies the embedded lib package into destDir/lib/
func extractLib(destDir string) error {
	return fs.WalkDir(libFS, "lib", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "lib" {
			return os.MkdirAll(filepath.Join(destDir, "lib"), 0755)
		}

		destPath := filepath.Join(destDir, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := libFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
}

// generateGoMod produces a go.mod with the module name "test" (matching the generated import "test/lib")
// and requires for whichever database drivers the job uses
func (j *JobExecution) generateGoMod() string {
	requires := make(map[string]string)

	// Always include fixed dependencies (e.g. NATS for progress reporting)
	for path, version := range fixedDependencies {
		requires[path] = version
	}

	// Add database driver dependencies based on connections used
	for _, conn := range j.Context.DBConnections {
		importPath := conn.GetImportPath()
		if version, ok := knownDriverVersions[importPath]; ok {
			requires[importPath] = version
		}
	}

	var b strings.Builder
	b.WriteString("module test\n\ngo 1.25.1\n")
	if len(requires) > 0 {
		b.WriteString("\nrequire (\n")
		for path, version := range requires {
			fmt.Fprintf(&b, "\t%s %s\n", path, version)
		}
		b.WriteString(")\n")
	}
	return b.String()
}

// dockerBuild builds a Docker image from the prepared workspace
func (j *JobExecution) dockerBuild(workDir, imageTag string) error {
	j.logger.Info().Msgf("Building Docker image: %s", imageTag)
	return pkg.RunCommandLine(workDir, "docker", "build", "-t", imageTag, ".")
}

// dockerRun executes the job inside a Docker container.
// In debug mode the container is kept alive so you can inspect it with:
//
//	docker logs <name>
//	docker cp <name>:/app /tmp/inspect
func (j *JobExecution) dockerRun(imageTag, containerName string) error {
	j.logger.Info().Msgf("Running job container: %s", containerName)
	args := []string{"run", "--network", "host", "--name", containerName}
	if !j.isDebug() {
		args = append(args, "--rm")
	}
	args = append(args, imageTag)
	return pkg.RunCommandLine("", "docker", args...)
}

// dockerRmi removes the Docker image after execution
func (j *JobExecution) dockerRmi(imageTag string) {
	if err := pkg.RunCommandLine("", "docker", "rmi", imageTag); err != nil {
		j.logger.Warn().Err(err).Msgf("Failed to remove image %s", imageTag)
	}
}

// newImageTag generates a unique Docker image tag for this job run
func (j *JobExecution) newImageTag() string {
	return fmt.Sprintf("job-%d-%s", j.Job.ID, uuid.NewString()[:8])
}

func (j *JobExecution) isDebug() bool {
	return api.GetEnv("RUN_MODE", "dev") == "dev"
}
