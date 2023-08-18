package helm

import (
	"context"

	"github.com/go-playground/validator/v10"
	v1 "github.com/wandb/operator/api/v1"
	"github.com/wandb/operator/pkg/wandb/spec"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RepoRelease struct {
	URL  string `validate:"required,url" json:"url"`
	Name string `validate:"required" json:"name"`

	// If version is not set, download latest.
	Version string `json:"version"`

	Password string `json:"password"`
	Username string `json:"username"`
}

func (r RepoRelease) Chart() (*chart.Chart, error) {
	local, err := r.ToLocalRelease()
	if err != nil {
		return nil, err
	}
	return local.Chart()
}

func (c RepoRelease) Validate() error {
	return validator.New().Struct(c)
}

func (r RepoRelease) ToLocalRelease() (*LocalRelease, error) {
	chartPath, err := r.downloadChart()
	if err != nil {
		return nil, err
	}

	local := new(LocalRelease)
	local.Path = chartPath
	return local, nil
}

func (r RepoRelease) Apply(
	ctx context.Context,
	c client.Client,
	wandb *v1.WeightsAndBiases,
	scheme *runtime.Scheme,
	config spec.Config,
) error {
	local, err := r.ToLocalRelease()
	if err != nil {
		return err
	}
	return local.Apply(ctx, c, wandb, scheme, config)
}

func (r RepoRelease) Prune(
	ctx context.Context,
	c client.Client,
	wandb *v1.WeightsAndBiases,
	scheme *runtime.Scheme,
	config spec.Config,
) error {
	local, err := r.ToLocalRelease()
	if err != nil {
		return err
	}
	return local.Prune(ctx, c, wandb, scheme, config)
}

func (r RepoRelease) downloadChart() (string, error) {
	entry := new(repo.Entry)
	entry.URL = r.URL
	entry.Name = r.Name
	entry.Username = r.Username
	entry.Password = r.Password

	file := repo.NewFile()
	file.Update(entry)

	settings := cli.New()
	providers := getter.All(settings)
	chartRepo, err := repo.NewChartRepository(entry, providers)
	if err != nil {
		return "", err
	}

	_, err = chartRepo.DownloadIndexFile()
	if err != nil {
		return "", err
	}

	chartURL, err := repo.FindChartInRepoURL(
		entry.URL, entry.Name, r.Version,
		"", "", "",
		providers,
	)
	if err != nil {
		return "", err
	}

	client := action.NewPull()
	client.Username = entry.Username
	client.Password = entry.Password
	client.Version = r.Version
	client.Settings = settings

	return client.Run(chartURL)
}
