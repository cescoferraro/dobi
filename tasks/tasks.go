package tasks

import (
	"fmt"
	"strings"

	"github.com/dnephin/dobi/config"
	"github.com/dnephin/dobi/logging"
	"github.com/dnephin/dobi/tasks/alias"
	"github.com/dnephin/dobi/tasks/compose"
	"github.com/dnephin/dobi/tasks/context"
	"github.com/dnephin/dobi/tasks/iface"
	"github.com/dnephin/dobi/tasks/image"
	"github.com/dnephin/dobi/tasks/mount"
	"github.com/dnephin/dobi/tasks/run"
	"github.com/dnephin/dobi/utils/stack"
	docker "github.com/fsouza/go-dockerclient"
)

// TaskCollection is a collection of Task objects
type TaskCollection struct {
	tasks []iface.Task
}

func (c *TaskCollection) add(task iface.Task, resource config.Resource) {
	c.tasks = append(c.tasks, task)
}

func (c *TaskCollection) contains(name string) bool {
	for _, task := range c.tasks {
		if task.Name() == name {
			return true
		}
	}
	return false
}

// All returns all the tasks in the dependency order
func (c *TaskCollection) All() []iface.Task {
	return c.tasks
}

// Reversed returns all the tasks in reversed dependency order
func (c *TaskCollection) Reversed() []iface.Task {
	tasks := []iface.Task{}
	for i := len(c.tasks) - 1; i >= 0; i-- {
		tasks = append(tasks, c.tasks[i])
	}
	return tasks
}

func newTaskCollection() *TaskCollection {
	return &TaskCollection{}
}

func collectTasks(options RunOptions) (*TaskCollection, error) {
	return collect(options, newTaskCollection(), stack.NewStringStack())
}

func collect(
	options RunOptions,
	tasks *TaskCollection,
	taskStack *stack.StringStack,
) (*TaskCollection, error) {
	for _, name := range options.Tasks {
		if tasks.contains(name) {
			continue
		}

		if taskStack.Contains(name) {
			return nil, fmt.Errorf(
				"Invalid dependency cycle: %s",
				strings.Join(taskStack.Items(), ", "))
		}

		name, action := splitAction(name)
		resource, ok := options.Config.Resources[name]
		if !ok {
			return nil, fmt.Errorf("Resource %q does not exist", name)
		}

		task, err := buildTaskFromResource(name, action, resource)
		if err != nil {
			return nil, err
		}
		taskStack.Push(name)
		options.Tasks = resource.Dependencies()
		if _, err := collect(options, tasks, taskStack); err != nil {
			return nil, err
		}
		tasks.add(task, resource)
		taskStack.Pop()
	}
	return tasks, nil
}

// TODO: some way to make this a registry
func buildTaskFromResource(name, action string, resource config.Resource) (iface.Task, error) {
	switch conf := resource.(type) {
	case *config.ImageConfig:
		return image.GetTask(name, action, conf)
	case *config.RunConfig:
		return run.GetTask(name, action, conf)
	case *config.MountConfig:
		return mount.GetTask(name, action, conf)
	case *config.AliasConfig:
		return alias.GetTask(name, action, conf)
	case *config.ComposeConfig:
		return compose.GetTask(name, action, conf)
	default:
		panic(fmt.Sprintf("Unexpected config type %T", conf))
	}

}

func splitAction(name string) (string, string) {
	parts := strings.SplitN(name, ":", 2)
	switch len(parts) {
	case 2:
		return parts[0], parts[1]
	default:
		return name, ""
	}
}

func executeTasks(ctx *context.ExecuteContext, tasks *TaskCollection) error {
	logging.Log.Debug("preparing tasks")
	for _, task := range tasks.All() {
		if err := task.Prepare(ctx); err != nil {
			return fmt.Errorf("Failed to prepare task %q: %s", task.Name(), err)
		}
	}

	defer func() {
		logging.Log.Debug("stopping tasks")
		for _, task := range tasks.Reversed() {
			if err := task.Stop(ctx); err != nil {
				logging.Log.Warnf("Failed to stop task %q: %s", task.Name(), err)
			}
		}
	}()

	logging.Log.Debug("executing tasks")
	for _, task := range tasks.All() {
		if err := task.Run(ctx); err != nil {
			return fmt.Errorf("Failed to execute task %q: %s", task.Name(), err)
		}
	}
	return nil
}

// RunOptions are the options supported by Run
type RunOptions struct {
	Client *docker.Client
	Config *config.Config
	Tasks  []string
	Quiet  bool
}

func getTaskNames(options RunOptions) []string {
	if len(options.Tasks) > 0 {
		return options.Tasks
	}

	if options.Config.Meta.Default != "" {
		return []string{options.Config.Meta.Default}
	}

	return options.Tasks
}

// Run one or more tasks
func Run(options RunOptions) error {
	options.Tasks = getTaskNames(options)
	if len(options.Tasks) == 0 {
		return fmt.Errorf("No task to run, and no default task defined.")
	}

	tasks, err := collectTasks(options)
	if err != nil {
		return err
	}

	execEnv, err := context.NewExecEnvFromConfig(options.Config)
	if err != nil {
		return err
	}

	ctx := context.NewExecuteContext(options.Config, options.Client, execEnv, options.Quiet)
	return executeTasks(ctx, tasks)
}
