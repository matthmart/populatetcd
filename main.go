package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
	"github.com/docker/engine-api/types/filters"
)

func main() {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}

	filters := filters.NewArgs()
	filters.Add("type", "network")

	options := types.EventsOptions{Filters: filters}
	reader, err := cli.Events(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	wg.Add(1)

	go func(reader io.Reader) {
		// _, err = io.Copy(os.Stdout, reader)
		// if err != nil && err != io.EOF {
		// 	log.Fatal(err)
		// }

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			data := new(events.Message)
			json.Unmarshal(scanner.Bytes(), data)
			fmt.Printf("Youpi ? %v \n\n\n", data.Actor.Attributes["container"])
		}
		// if err := scanner.Err(); err != nil {
		// 	fmt.Fprintln(os.Stderr, "There was an error with the scanner in attached container", err)
		// }
	}(reader)

	wg.Wait()
}
