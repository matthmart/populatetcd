package utils

import (
	"log"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"golang.org/x/net/context"
)

func GetContainersList(ctx context.Context, dockerCli *client.Client) ([]types.Container, error) {
	filters := filters.NewArgs()
	filters.Add("label", "proxy.domain_names")

	containers, err := dockerCli.ContainerList(ctx, types.ContainerListOptions{Filter: filters})
	if err != nil {
		log.Println("Unable to fetch containers: ", err)
		return []types.Container{}, err
	}

	return containers, nil
}
