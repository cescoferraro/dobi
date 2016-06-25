package tasks

import (
	"testing"

	"github.com/dnephin/dobi/config"
	"github.com/stretchr/testify/assert"
)

func TestPrepareTasksErrorsOnCyclicDependencies(t *testing.T) {
	runOptions := RunOptions{
		Config: &config.Config{
			Resources: map[string]config.Resource{
				"one":   &config.ImageConfig{Depends: []string{"two"}},
				"two":   &config.ImageConfig{Depends: []string{"three"}},
				"three": &config.ImageConfig{Depends: []string{"four", "one"}},
				"four":  &config.ImageConfig{Depends: []string{"five"}},
				"five":  &config.ImageConfig{},
			},
		},
		Tasks: []string{"one"},
	}
	tasks, err := prepareTasks(runOptions)
	assert.Nil(t, tasks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid dependency cycle: one, two, three")
}

func TestPrepareTasksDoesNotErrorOnDuplicateTask(t *testing.T) {
	runOptions := RunOptions{
		Config: &config.Config{
			Resources: map[string]config.Resource{
				"one": &config.ImageConfig{},
				"two": &config.ImageConfig{Depends: []string{"one"}},
			},
		},
		Tasks: []string{"one", "two"},
	}
	tasks, err := prepareTasks(runOptions)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(tasks.tasks))
}