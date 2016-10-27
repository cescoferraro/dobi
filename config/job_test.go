package config

import (
	"reflect"
	"testing"

	pth "github.com/dnephin/configtf/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type JobConfigSuite struct {
	suite.Suite
	job  *JobConfig
	conf *Config
}

func TestJobConfigSuite(t *testing.T) {
	suite.Run(t, new(JobConfigSuite))
}

func (s *JobConfigSuite) SetupTest() {
	s.job = &JobConfig{}
	s.conf = NewConfig()
}

func (s *JobConfigSuite) TestString() {
	s.job.Use = "builder"
	s.job.Command = ShlexSlice{original: "run"}
	s.job.Artifact = "foo"
	s.Equal(s.job.String(), "Run 'run' using the 'builder' image to create 'foo'")
}

func (s *JobConfigSuite) TestValidateMissingUse() {
	s.conf.Resources["example"] = &AliasConfig{}
	s.job.Use = "example"
	err := s.job.Validate(pth.NewPath(""), s.conf)
	s.Error(err)
	s.Contains(err.Error(), "example is not an image resource")
}

func (s *JobConfigSuite) TestValidateMissingMount() {
	s.conf.Resources["one"] = NewImageConfig()
	s.conf.Resources["two"] = NewImageConfig()
	s.conf.Resources["example"] = NewImageConfig()
	s.job.Use = "example"
	s.job.Mounts = []string{"one", "two"}

	err := s.job.Validate(pth.NewPath(""), s.conf)
	s.Error(err)
	s.Contains(err.Error(), "one is not a mount resource")
}

func (s *JobConfigSuite) TestRunFromConfig() {
	values := map[string]interface{}{
		"use":        "image-res",
		"command":    "echo foo",
		"entrypoint": "bash -c",
	}
	res, err := jobFromConfig("foo", values)
	job, ok := res.(*JobConfig)

	s.Equal(ok, true)
	s.Nil(err)
	s.Equal(job.Use, "image-res")
	s.Equal(job.Command.Value(), []string{"echo", "foo"})
	s.Equal(job.Entrypoint.Value(), []string{"bash", "-c"})
}

func TestShlexSliceTransformConfig(t *testing.T) {
	s := ShlexSlice{}
	zero := reflect.Value{}
	err := s.TransformConfig(zero)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}
