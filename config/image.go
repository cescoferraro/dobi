package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dnephin/configtf"
	pth "github.com/dnephin/configtf/path"
	"github.com/dnephin/dobi/execenv"
	docker "github.com/fsouza/go-dockerclient"
)

// ImageConfig An **image** resource provides actions for working with a Docker
// image. If an image is buildable it is considered up-to-date if all files in
// the build context have a modified time older than the created time of the
// image.
// name: image
// example: An image with build args:
//
// .. code-block:: yaml
//
//     image=project-dev:
//         image: myproject-dev
//         context: .
//         args:
//           version: '3.1.4'
//           url: http://example.com/foo
//
type ImageConfig struct {
	// Image The name of the **image** without a tag. Tags must be specified
	// in the **tags** field.
	Image string `config:"required,validate"`
	// Dockerfile The path to the ``Dockerfile`` used to build the image. This
	// path is relative to the **context**.
	Dockerfile string
	// Context The build context used to build the image.
	// default: ``.``
	Context string
	// Args Build args used to build the image. Values in the mapping support
	// :doc:`variables`.
	// type: mapping ``key: value``
	Args map[string]string
	// PullBaseImageOnBuild If **true** the base image used in the
	// ``Dockerfile`` will be pulled before building the image.
	PullBaseImageOnBuild bool
	// Pull Pull an image instead of building it. The value may be one of:
	// * ``once`` - only pull if the image:tag does not exist
	// * ``always`` - always pull the image
	// * ``<duration>`` - pull if the image hasn't been pulled in at least
	//   ``duration``. The format of duration is a number followed by a single
	//   character time unit (ex: ``40s``, ``2h``, ``30min``)
	// type: string
	// default: ``always``
	Pull pull
	// Tags The image tags applied to the image before pushing the image to a
	// registry.  The first tag in the list is used when the image is built.
	// Each item in the list supports :doc:`variables`.
	// default: ``['{unique}']``
	// type: list of tags
	Tags []string
	dependent
	describable
}

// Validate checks that all fields have acceptable values
func (c *ImageConfig) Validate(path pth.Path, config *Config) *pth.Error {
	if err := c.validateBuildOrPull(); err != nil {
		return pth.Errorf(path, err.Error())
	}
	return nil
}

func (c *ImageConfig) validateBuildOrPull() error {
	if c.Dockerfile == "" && c.Context == "" && !c.Pull.IsSet() {
		return fmt.Errorf("one of dockerfile, context, or pull is required")
	}
	switch {
	case c.Dockerfile == "" && c.Context != "":
		c.Dockerfile = "Dockerfile"
	case c.Context == "" && c.Dockerfile != "":
		c.Context = "."
	}
	return nil
}

// ValidateImage validates the image field does not include a tag
func (c *ImageConfig) ValidateImage() error {
	_, tag := docker.ParseRepositoryTag(c.Image)
	if tag != "" {
		return fmt.Errorf(
			"Tag %q must be specified in the `tags` field, not in `image`", tag)
	}
	return nil
}

func (c *ImageConfig) String() string {
	dir := filepath.Join(c.Context, c.Dockerfile)
	return fmt.Sprintf("Build image '%s' from '%s'", c.Image, dir)
}

// Resolve resolves variables in the resource
func (c *ImageConfig) Resolve(env *execenv.ExecEnv) (Resource, error) {
	var err error
	c.Tags, err = env.ResolveSlice(c.Tags)
	if err != nil {
		return c, err
	}

	for key, value := range c.Args {
		c.Args[key], err = env.Resolve(value)
		if err != nil {
			return c, err
		}
	}
	return c, nil
}

// NewImageConfig creates a new ImageConfig with default values
func NewImageConfig() *ImageConfig {
	return &ImageConfig{}
}

type pullAction func(*time.Time) bool

type pull struct {
	action pullAction
}

func (p *pull) TransformConfig(raw reflect.Value) error {
	if !raw.IsValid() {
		return fmt.Errorf("must be a string, was undefined")
	}

	switch value := raw.Interface().(type) {
	case string:
		switch value {
		case "once":
			p.action = pullOnce
		case "always":
			p.action = pullAlways
		default:
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid pull value %q: %s", value, err)
			}
			p.action = pullAfter{duration: duration}.doPull
		}
	default:
		return fmt.Errorf("must be a string, not %T", value)
	}
	return nil
}

func (p *pull) Required(lastPull *time.Time) bool {
	if !p.IsSet() {
		return true
	}
	return p.action(lastPull)
}

func (p *pull) IsSet() bool {
	return p.action != nil
}

func pullAlways(lastPull *time.Time) bool {
	return true
}

func pullOnce(lastPull *time.Time) bool {
	return lastPull == nil
}

type pullAfter struct {
	duration time.Duration
}

func (p pullAfter) doPull(lastPull *time.Time) bool {
	if lastPull == nil {
		return true
	}
	return lastPull.Before(time.Now().Add(-p.duration))
}

func imageFromConfig(name string, values map[string]interface{}) (Resource, error) {
	image := NewImageConfig()
	return image, configtf.Transform(name, values, image)
}

func init() {
	RegisterResource("image", imageFromConfig)
}
