package steps

import (
	"context"
	"fmt"
	"strconv"

	buildapi "github.com/openshift/api/build/v1"
	"github.com/openshift/ci-operator/pkg/api"
	imageclientset "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
)

func rawCommandDockerfile(from api.PipelineImageStreamTagReference, commands string) string {
	return fmt.Sprintf(`FROM %s:%s
RUN ["/bin/bash", "-c", %s]`, PipelineImageStream, from, strconv.Quote(fmt.Sprintf("set -o errexit; umask 0002; %s", commands)))
}

type pipelineImageCacheStep struct {
	config      api.PipelineImageCacheStepConfiguration
	buildClient BuildClient
	istClient   imageclientset.ImageStreamTagsGetter
	jobSpec     *JobSpec
}

func (s *pipelineImageCacheStep) Inputs(ctx context.Context, dry bool) (api.InputDefinition, error) {
	return nil, nil
}

func (s *pipelineImageCacheStep) Run(ctx context.Context, dry bool) error {
	dockerfile := rawCommandDockerfile(s.config.From, s.config.Commands)
	return handleBuild(s.buildClient, buildFromSource(
		s.jobSpec, s.config.From, s.config.To,
		buildapi.BuildSource{
			Type:       buildapi.BuildSourceDockerfile,
			Dockerfile: &dockerfile,
		},
	), dry)
}

func (s *pipelineImageCacheStep) Done() (bool, error) {
	return imageStreamTagExists(s.config.To, s.istClient.ImageStreamTags(s.jobSpec.Namespace()))
}

func (s *pipelineImageCacheStep) Requires() []api.StepLink {
	return []api.StepLink{api.InternalImageLink(s.config.From)}
}

func (s *pipelineImageCacheStep) Creates() []api.StepLink {
	return []api.StepLink{api.InternalImageLink(s.config.To)}
}

func (s *pipelineImageCacheStep) Provides() (api.ParameterMap, api.StepLink) {
	return nil, nil
}

func (s *pipelineImageCacheStep) Name() string { return string(s.config.To) }

func PipelineImageCacheStep(config api.PipelineImageCacheStepConfiguration, buildClient BuildClient, istClient imageclientset.ImageStreamTagsGetter, jobSpec *JobSpec) api.Step {
	return &pipelineImageCacheStep{
		config:      config,
		buildClient: buildClient,
		istClient:   istClient,
		jobSpec:     jobSpec,
	}
}
