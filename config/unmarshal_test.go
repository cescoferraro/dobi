package config

import (
	"testing"

	"github.com/renstrom/dedent"
	"github.com/stretchr/testify/assert"
)

func TestLoadFromBytes(t *testing.T) {
	conf := dedent.Dedent(`
		meta:
		  default: alias-def

		image=image-def:
		  image: imagename
		  dockerfile: what
		  args:
		    VERSION: "3.3.3"
		    DEBUG: 'true'

		mount=vol-def:
		  bind: dist/
		  path: /target

		job=cmd-def:
		  use: image-dev
		  mounts: [vol-def]

		alias=alias-def:
		  tasks: [vol-def, cmd-def]

		compose=compose-def:
		  files: ['foo.yml']
	`)

	config, err := LoadFromBytes([]byte(conf))
	assert.Nil(t, err)
	assert.Equal(t, 5, len(config.Resources))
	assert.IsType(t, &ImageConfig{}, config.Resources["image-def"])
	assert.IsType(t, &MountConfig{}, config.Resources["vol-def"])
	assert.IsType(t, &JobConfig{}, config.Resources["cmd-def"])
	assert.IsType(t, &AliasConfig{}, config.Resources["alias-def"])

	// Test default value and override
	imageConf := config.Resources["image-def"].(*ImageConfig)
	assert.Equal(t, "what", imageConf.Dockerfile)
	assert.Equal(t, map[string]string{
		"VERSION": "3.3.3",
		"DEBUG":   "true",
	}, imageConf.Args)

	mountConf := config.Resources["vol-def"].(*MountConfig)
	assert.Equal(t, "dist/", mountConf.Bind)
	assert.Equal(t, "/target", mountConf.Path)
	assert.Equal(t, false, mountConf.ReadOnly)

	aliasConf := config.Resources["alias-def"].(*AliasConfig)
	assert.Equal(t, []string{"vol-def", "cmd-def"}, aliasConf.Tasks)

	assert.Equal(t, &MetaConfig{Default: "alias-def"}, config.Meta)
}

func TestLoadFromBytesWithReservedName(t *testing.T) {
	conf := dedent.Dedent(`
		image=image-def:
		  image: imagename
		  dockerfile: what

		mount=autoclean:
		  path: dist/
		  mount: /target
	`)

	_, err := LoadFromBytes([]byte(conf))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "\"autoclean\" is reserved")
}

func TestLoadFromBytesWithInvalidName(t *testing.T) {
	conf := dedent.Dedent(`
		image=image:latest:
		  image: imagename
		  dockerfile: what
	`)

	_, err := LoadFromBytes([]byte(conf))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character \":\"")
}
