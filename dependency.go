package dev

import (
	"github.com/goombaio/dag"
	d "github.com/goombaio/dag"
	"github.com/pkg/errors"

	c "github.com/wish/dev/config"

	log "github.com/sirupsen/logrus"
)

const (
	// BUILD constant referring to the build command of this project which
	// builds the project with docker-compose as specified in this tools
	// configuration file.
	BUILD = "build"
	// DOWN constant referring to the "down" command of this project which
	// stops and removes the project container.
	DOWN = "down"
	// PS constant referring to the "ps" command of this project which
	// shows the status of the containers used by the project.
	PS = "ps"
	// SH constant referring to the "sh" command of this project which runs
	// commands on the project container.
	SH = "sh"
	// UP constant referring to the "up" command of this project which
	// starts the project and any of the specified dependencies.
	UP = "up"
)

// Dependency is the interface that is used by all objects in the dev
// configuration implement such that they can be used as a dependency by other
// objects of the configuration.
type Dependency interface {
	// PreRun does whatever is required of the dependency. It is run prior
	// to the specified command for the given project.
	PreRun(command string, appConfig *c.Dev, project *Project)
	// Dependencies returns the names of all the dev objects it depends on
	// in order to function.
	Dependencies() []string
	// Name of the depencency. Maps to the name given to the object in the
	// dev configuration file.
	GetName() string
}

func createObjectMap(devConfig *c.Dev) map[string]Dependency {
	objMap := make(map[string]Dependency)

	for name, opts := range devConfig.Projects {
		objMap[name] = NewProject(opts)
	}

	for name, opts := range devConfig.Networks {
		objMap[name] = NewNetwork(name, opts)
	}

	for name, opts := range devConfig.Registries {
		objMap[name] = NewRegistry(opts)
	}

	return objMap
}

func addDeps(objMap map[string]Dependency, dag *d.DAG, obj Dependency) error {
	for _, depName := range obj.Dependencies() {
		vertex := d.NewVertex(depName, objMap[depName])
		if err := dag.AddVertex(vertex); err != nil {
			return err
		}
		parent, err := dag.GetVertex(obj.GetName())
		if err != nil {
			return errors.Wrapf(err, "Unable to locate vertex for: %s", obj.GetName())
		}
		if err := dag.AddEdge(parent, vertex); err != nil {
			return errors.Wrapf(err, "Failure adding edge from %s to %s", parent.ID, vertex.ID)
		}

		if err := addDeps(objMap, dag, objMap[depName]); err != nil {
			return err
		}
	}
	return nil
}

// Hmm, the libraries deleteEdge does not remove the parent from the childs
// parent list.
func deleteEdge(parent *dag.Vertex, child *dag.Vertex) error {
	for _, c := range parent.Children.Values() {
		if c == child {
			parent.Children.Remove(child)
			child.Parents.Remove(parent)
		}
	}

	return nil
}

func topologicalSort(dag *d.DAG, vertex *d.Vertex) ([]string, error) {
	sorted := []string{}
	parentless := make(map[string]bool)

	parentless[vertex.ID] = true

	for ok := true; ok; ok = len(parentless) > 0 {
		var n string
		for key := range parentless {
			n = key
			break
		}
		sorted = SliceInsertString(sorted, n, 0)
		delete(parentless, n)

		v, err := dag.GetVertex(n)
		if err != nil {
			return nil, errors.Wrapf(err, "Unexpected missing vertex for: %s", n)
		}

		children, _ := dag.Successors(v)
		for _, child := range children {
			//log.Debugf("got a child %s with %d incoming edges", child.ID, child.InDegree())
			if err := deleteEdge(v, child); err != nil {
				return nil, err
			}
			if child.InDegree() == 0 {
				//log.Debugf("no incoming edges for %s", child.ID)
				parentless[child.ID] = true
			}
		}
	}

	if dag.Size() != 0 {
		return nil, errors.Errorf("Dependency graph has a cycle")
	}

	// do not return the initial vertex in the final list, b/c we are only
	// interested in the dependencies
	return sorted[0 : len(sorted)-1], nil
}

// InitDeps runs the PreRun method on each dependency for the specified
// Project.
func InitDeps(appConfig *c.Dev, cmd string, project *Project) error {
	dag := d.NewDAG()
	vertex := d.NewVertex(project.Name, project)
	if err := dag.AddVertex(vertex); err != nil {
		return err
	}

	objMap := createObjectMap(appConfig)
	if err := addDeps(objMap, dag, project); err != nil {
		return errors.Wrap(err, "Failure mapping dependencies")
	}

	deps, err := topologicalSort(dag, vertex)
	if err != nil {
		return errors.Wrapf(err, "Failure sorting dependencies for %s", project.Name)
	}

	log.Debugf("Initializing dependencies for %s: %s", project.Name, deps)
	for _, dep := range deps {
		objMap[dep].PreRun(cmd, appConfig, project)
	}

	return nil
}
