package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
)

func main() {

	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
		log.Println("Program killed !")

		// do last actions and wait for all write operations to end

		os.Exit(0)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dockerCli, err := client.NewEnvClient()
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
	etcdCli := etcd.NewKeysAPI(c)
	fmt.Println("etcd API connection OK.")

	// updateConfig(ctx, dockerCli, etcdCli)
	listenForEvents(ctx, dockerCli, etcdCli)
}

func listenForEvents(ctx context.Context, dockerCli *client.Client, etcdCli etcd.KeysAPI) {
	filters := filters.NewArgs()
	filters.Add("type", "network")
	// filters.Add("network", "hae_default")

	options := types.EventsOptions{Filters: filters}
	reader, err := dockerCli.Events(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	fmt.Println("Listening for events...")

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {

		fmt.Println(scanner.Text())

		updateConfig(ctx, dockerCli, etcdCli)

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
}

func updateConfig(ctx context.Context, dockerCli *client.Client, etcdCli etcd.KeysAPI) {

	filters := filters.NewArgs()
	// filters.Add("label", "com.docker.compose.project=hae")

	containers, err := dockerCli.ContainerList(ctx, types.ContainerListOptions{Filter: filters})
	if err != nil {
		log.Println("Unable to fetch containers: ", err)
		return
	}

	// clear content
	etcdCli.Delete(ctx, "/subproxies", &etcd.DeleteOptions{Recursive: true})

	for _, container := range containers {
		if hosts, ok := container.Labels["proxy.domain_names"]; ok {
			fmt.Println(hosts)
			etcdCli.Set(ctx, "/subproxies"+container.Names[0]+"/hosts", hosts, nil)
		}
	}
}
