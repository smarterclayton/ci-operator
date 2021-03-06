package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	imageapi "github.com/openshift/api/image/v1"
	"github.com/openshift/ci-operator/pkg/api"
	imageclientset "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// outputImageTagStep will ensure that a tag exists
// in the named ImageStream that resolves to the built
// pipeline image
type outputImageTagStep struct {
	config    api.OutputImageTagStepConfiguration
	istClient imageclientset.ImageStreamTagsGetter
	isClient  imageclientset.ImageStreamsGetter
	jobSpec   *JobSpec
}

func (s *outputImageTagStep) Inputs(ctx context.Context, dry bool) (api.InputDefinition, error) {
	return nil, nil
}

func (s *outputImageTagStep) Run(ctx context.Context, dry bool) error {
	log.Printf("Tagging %s/%s:%s into %s:%s", s.jobSpec.Namespace(), PipelineImageStream, s.config.From, s.config.To.Name, s.config.To.Tag)
	fromImage := "dry-fake"
	if !dry {
		from, err := s.istClient.ImageStreamTags(s.jobSpec.Namespace()).Get(fmt.Sprintf("%s:%s", PipelineImageStream, s.config.From), meta.GetOptions{})
		if err != nil {
			return fmt.Errorf("could not resolve base image: %v", err)
		}
		fromImage = from.Image.Name
	}
	ist := &imageapi.ImageStreamTag{
		ObjectMeta: meta.ObjectMeta{
			Name:      fmt.Sprintf("%s:%s", s.config.To.Name, s.config.To.Tag),
			Namespace: s.jobSpec.Namespace(),
		},
		Tag: &imageapi.TagReference{
			ReferencePolicy: imageapi.TagReferencePolicy{
				Type: imageapi.LocalTagReferencePolicy,
			},
			From: &coreapi.ObjectReference{
				Kind:      "ImageStreamImage",
				Name:      fmt.Sprintf("%s@%s", PipelineImageStream, fromImage),
				Namespace: s.jobSpec.Namespace(),
			},
		},
	}
	if dry {
		istJSON, err := json.Marshal(ist)
		if err != nil {
			return fmt.Errorf("failed to marshal imagestreamtag: %v", err)
		}
		fmt.Printf("%s\n", istJSON)
	} else {
		if err := s.istClient.ImageStreamTags(s.jobSpec.Namespace()).Delete(ist.Name, nil); err != nil && !errors.IsNotFound(err) {
			return err
		}
		_, err := s.istClient.ImageStreamTags(s.jobSpec.Namespace()).Create(ist)
		if errors.IsAlreadyExists(err) {
			// another job raced with us, but the end
			// result will be the same so we don't care
			return nil
		}
		return err
	}

	return nil
}

func (s *outputImageTagStep) Done() (bool, error) {
	log.Printf("Checking for existence of %s:%s", PipelineImageStream, s.config.To)
	_, err := s.istClient.ImageStreamTags(s.jobSpec.Namespace()).Get(
		fmt.Sprintf("%s:%s", PipelineImageStream, s.config.To),
		meta.GetOptions{},
	)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return true, nil
	}
}

func (s *outputImageTagStep) Requires() []api.StepLink {
	return []api.StepLink{api.InternalImageLink(s.config.From), api.ReleaseImagesLink()}
}

func (s *outputImageTagStep) Creates() []api.StepLink {
	if len(s.config.To.As) > 0 {
		return []api.StepLink{api.ExternalImageLink(s.config.To), api.InternalImageLink(api.PipelineImageStreamTagReference(s.config.To.As))}
	}
	return []api.StepLink{api.ExternalImageLink(s.config.To)}
}

func (s *outputImageTagStep) Provides() (api.ParameterMap, api.StepLink) {
	if len(s.config.To.As) == 0 {
		return nil, nil
	}
	return api.ParameterMap{
		fmt.Sprintf("IMAGE_%s", strings.ToUpper(strings.Replace(s.config.To.As, "-", "_", -1))): func() (string, error) {
			is, err := s.isClient.ImageStreams(s.jobSpec.Namespace()).Get(s.config.To.Name, meta.GetOptions{})
			if err != nil {
				return "", err
			}
			var registry string
			if len(is.Status.PublicDockerImageRepository) > 0 {
				registry = is.Status.PublicDockerImageRepository
			} else if len(is.Status.DockerImageRepository) > 0 {
				registry = is.Status.DockerImageRepository
			} else {
				return "", fmt.Errorf("image stream %s has no accessible image registry value", s.config.To.As)
			}
			return fmt.Sprintf("%s:%s", registry, s.config.To.Tag), nil
		},
	}, api.ExternalImageLink(s.config.To)
}

func (s *outputImageTagStep) Name() string { return s.config.To.As }

func OutputImageTagStep(config api.OutputImageTagStepConfiguration, istClient imageclientset.ImageStreamTagsGetter, isClient imageclientset.ImageStreamsGetter, jobSpec *JobSpec) api.Step {
	return &outputImageTagStep{
		config:    config,
		istClient: istClient,
		isClient:  isClient,
		jobSpec:   jobSpec,
	}
}
