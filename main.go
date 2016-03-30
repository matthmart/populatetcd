package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
)

func main() {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cli, err := client.NewEnvClient()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Docker API connection OK.")

		cfg := etcd.Config{
			Endpoints: []string{"http://192.168.99.100:2379"},
			Transport: etcd.DefaultTransport,
			// set timeout per request to fail fast when the target endpoint is unavailable
			HeaderTimeoutPerRequest: time.Second,
		}
		c, err := etcd.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		kapi := etcd.NewKeysAPI(c)
		fmt.Println("etcd API connection OK.")

		filters := filters.NewArgs()
		filters.Add("type", "network")
		// filters.Add("network", "hae_default")

		options := types.EventsOptions{Filters: filters}
		reader, err := cli.Events(ctx, options)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()

		fmt.Println("Listening for events...")

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {

			fmt.Println(scanner.Text())

			updateConfig(cli, ctx, kapi)

			// data := new(events.Message)
			// json.Unmarshal(scanner.Bytes(), data)

			// containerID := data.Actor.Attributes["container"]
			// fmt.Printf("Network state changed for %v\n", containerID)
			//
			// containerInfo, err := cli.ContainerInspect(ctx, containerID)
			// if err != nil {
			// 	fmt.Fprintln(os.Stderr, "Inspect error", err)
			// }
			//
			// fmt.Println("## Infos ##")
			// fmt.Println("  name: ", containerInfo.Name)
			// fmt.Println("  labels: ", containerInfo.Config.Labels)

		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "There was an error with the scanner", err)
		}

	}()

	wg.Wait()
}

func updateConfig(cli *client.Client, ctx context.Context, kapi etcd.KeysAPI) {

	filters := filters.NewArgs()
	// filters.Add("label", "com.docker.compose.project=hae")

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filter: filters})
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(containers)

	// clear content
	kapi.Delete(ctx, "/subproxies", &etcd.DeleteOptions{Recursive: true})

	for _, container := range containers {
		if hosts, ok := container.Labels["proxy.domain_names"]; ok {
			fmt.Println(hosts)
			kapi.Set(ctx, "/subproxies"+container.Names[0]+"/hosts", hosts, nil)
		}
	}
}
