package compose

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	errors "github.com/pkg/errors"
)

func buildConfigDetails(dir string, source map[string]interface{}) *types.ConfigDetails {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return &types.ConfigDetails{
		WorkingDir: workingDir,
		ConfigFiles: []types.ConfigFile{
			{Filename: "filename.yml", Config: source},
		},
		Environment: map[string]string{
			"HOME": os.Getenv("HOME"),
		},
	}
}

// Parse reads and parses the specified docker-compose.yml files and returns
// a map holdind the parsed structure representing each file.
func Parse(wd string, file string) (*types.Config, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not read %s", file)
	}
	dict, err := loader.ParseYAML(b)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing %s", file)
	}

	details := buildConfigDetails(filepath.Dir(file), dict)
	config, err := loader.Load(*details)
	if err != nil {
		return nil, errors.Wrapf(err, "Error loading %s", file)
	}

	return config, nil
}
