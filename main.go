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
	"github.com/jawher/mow.cli"
)

func main() {

	app := cli.App("populatetcd", "Update etcd according to containers labels")

	app.Spec = "[-d|--daemon]"

	daemon := app.BoolOpt("d daemon", false, "Run populatetcd in daemon mode, listening docker events")

	app.Action = func() {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dockerCli, err := client.NewEnvClient()
		if err != nil {
			log.Fatal(err)
		}

		// check if the Docker API is ok
		_, err = dockerCli.ServerVersion(ctx)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Docker API connection OK.")

		cfg := etcd.Config{
			Endpoints: []string{"http://192.168.99.100:2379"},
			Transport: etcd.DefaultTransport,
			// set timeout per request to fail fast when the target endpoint is unavailable
			HeaderTimeoutPerRequest: 3 * time.Second,
		}
		c, err := etcd.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		etcdCli := etcd.NewKeysAPI(c)

		if *daemon {
			done := make(chan bool, 1)
			// go listenForEvents(done, ctx, dockerCli, etcdCli)
			go polling(2*time.Second, done, ctx, dockerCli, etcdCli)
			<-done
		} else {
			updateConfig(ctx, dockerCli, etcdCli)
		}
	}

	app.Run(os.Args)
}

func polling(interval time.Duration, done chan<- bool, ctx context.Context, dockerCli *client.Client, etcdCli etcd.KeysAPI) {

	ticker := time.NewTicker(interval)

	// handle the SIGINT signal
	listening := make(chan os.Signal, 1)
	signal.Notify(listening, os.Interrupt)

	go func() {
		for range ticker.C {
			updateConfig(ctx, dockerCli, etcdCli)
		}
	}()

	// waiting for events
	<-listening

	ticker.Stop()
	fmt.Println("\n Exiting.")

	done <- true
}

func listenForEvents(done chan<- bool, ctx context.Context, dockerCli *client.Client, etcdCli etcd.KeysAPI) {
	filters := filters.NewArgs()
	filters.Add("type", "network")
	filters.Add("event", "connect")
	filters.Add("event", "disconnect")
	// filters.Add("network", "hae_default")

	options := types.EventsOptions{Filters: filters}
	reader, err := dockerCli.Events(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening for events...\n\n")

	// handle the SIGINT signal
	listening := make(chan os.Signal, 1)
	signal.Notify(listening, os.Interrupt)

	go func(listening chan os.Signal) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {

			// fmt.Println(scanner.Text())
			// log.Println("Network event received.")

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

		// handling scanner error
		if err := scanner.Err(); err != nil {
			fmt.Println("There was an error with the scanner", err)
			listening <- os.Interrupt
		}
	}(listening)

	// waiting for events
	<-listening

	reader.Close()
	done <- true
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
	// etcdCli.Delete(ctx, "/subproxies", &etcd.DeleteOptions{Recursive: true})
	// if err != nil {
	// 	log.Println("Unable to update etcd: ", err)
	// }

	fmt.Println("#### ectd ####")
	resp, err := etcdCli.Get(ctx, "/subproxies", nil)
	if resp != nil && resp.Node != nil {
		for _, subproxy := range resp.Node.Nodes {
			fmt.Println(subproxy.Key)
		}
		fmt.Println("##############")
	}

	for _, container := range containers {
		if hosts, ok := container.Labels["proxy.domain_names"]; ok {

			log.Println(container.Names[0], " domain_names: ", hosts)

			_, err = etcdCli.Set(ctx, "/subproxies"+container.Names[0]+"/hosts", hosts, nil)
			if err != nil {
				log.Println("Unable to update etcd: ", err)
				return
			}
		}
	}

	// log.Println("etcd updated.")
}
