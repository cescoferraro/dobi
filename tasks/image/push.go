package image

import (
	"io"
	"os"

	"github.com/dnephin/dobi/tasks/context"
	docker "github.com/fsouza/go-dockerclient"
)

// RunPush pushes an image to the registry
func RunPush(ctx *context.ExecuteContext, t *Task, _ bool) (bool, error) {
	pushTag := func(tag string) error {
		return pushImage(ctx, t, tag)
	}
	if err := t.ForEachTag(ctx, pushTag); err != nil {
		return false, err
	}
	t.logger().Info("Pushed")
	return true, nil
}

func pushImage(ctx *context.ExecuteContext, t *Task, tag string) error {
	repo, err := parseAuthRepo(t.config.Image)
	if err != nil {
		return err
	}

	return Stream(os.Stdout, func(out io.Writer) error {
		return ctx.Client.PushImage(docker.PushImageOptions{
			Name:          tag,
			OutputStream:  out,
			RawJSONStream: true,
			// TODO: timeout
		}, ctx.GetAuthConfig(repo))
	})
}
