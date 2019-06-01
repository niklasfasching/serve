package main

import (
	"log"
	"os"
	"syscall"

	"github.com/niklasfasching/serve"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("USAGE: %s {path_to_config.json}", os.Args[0])
	}
	for {
		ctx := serve.ReloadSignalContext(syscall.SIGUSR1)
		config, err := serve.ReadConfig(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		if err := serve.Start(ctx, config); err != nil {
			log.Fatal(err)
		}
	}
}
