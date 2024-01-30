package cleanup

import (
	"context"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/weka/gohomecli/internal/local/bundle"
	"golang.org/x/exp/slices"
)

const kubernetesImagesNamespace = "k8s.io"

type ImagesArgs struct {
	Force  bool
	DryRun bool
}

func Images(ctx context.Context, args ImagesArgs) error {
	logger.Info().Msg("Removing unused images")

	client, err := containerd.New("/run/k3s/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	imgs, err := getImages(ctx, client)
	if err != nil {
		return err
	}

	inUse, err := getInUseImages(ctx, client)
	if err != nil {
		return err
	}

	for _, img := range imgs {
		if inUse[img] {
			logger.Debug().Str("image", img).Msg("Skipping in-use image")
			continue
		}

		if additionallyProtected(img) && !args.Force {
			logger.Debug().Str("image", img).Msg("Skipping additionally protected image")
			continue
		}

		logger.Info().Str("image", img).Msg("Removing image")

		if !args.DryRun {
			err = client.ImageService().Delete(ctx, img)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getImages(ctx context.Context, client *containerd.Client) ([]string, error) {
	list, err := client.ImageService().List(namespaces.WithNamespace(ctx, kubernetesImagesNamespace))
	if err != nil {
		return nil, err
	}

	var imgs []string

	for _, img := range list {
		logger.Debug().Interface("image", img).Msg("Got image")
		imgs = append(imgs, img.Target.Annotations[images.AnnotationImageName])
	}

	slices.Sort(imgs)

	return slices.Compact(imgs), nil
}

func getInUseImages(ctx context.Context, client *containerd.Client) (map[string]bool, error) {
	var protected = map[string]bool{}

	ns, err := client.NamespaceService().List(ctx)
	if err != nil {
		return nil, err
	}

	// getting protected from runtime
	for _, n := range ns {
		logger.Debug().Str("namespace", n).Msg("Getting containers")
		containers, err := client.ContainerService().List(namespaces.WithNamespace(ctx, n))
		if err != nil {
			return nil, err
		}

		for _, container := range containers {
			logger.Debug().Str("namespace", n).
				Str("image", container.Image).
				Interface("labels", container.Labels).
				Msg("Got container image")
			protected[container.Image] = true
		}
	}

	// additionally protected images from bundle
	imgs, err := bundle.Images()
	if err != nil {
		return nil, err
	}

	for _, img := range imgs {
		protected[img.Name] = true
	}

	return protected, nil
}

var protectedPrefixes = []string{
	"docker.io/rancher/mirrored-library-busybox", // was used for k3s setup, keeping just in case
}

func additionallyProtected(image string) bool {
	for i := range protectedPrefixes {
		if strings.HasPrefix(image, protectedPrefixes[i]) {
			return true
		}
	}

	return false
}
