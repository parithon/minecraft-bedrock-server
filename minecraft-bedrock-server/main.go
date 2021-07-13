package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fmorenovr/gotry"
	"github.com/parithon/minecraft-bedrock-daemon/minecraft-bedrock-server/management"
	"github.com/parithon/minecraft-bedrock-daemon/minecraft-bedrock-server/minecraft"
	"github.com/parithon/minecraft-bedrock-daemon/minecraft-bedrock-server/utils"
)

func main() {

	var shutdown bool
	var terminate bool
	var healthcheck bool

	flag.BoolVar(&shutdown, "shutdown", false, "Shutdown this server after 30 seconds")
	flag.BoolVar(&terminate, "terminate", false, "Terminate the server immediately")
	flag.BoolVar(&healthcheck, "healthcheck", false, "Perform a health check on the server")

	flag.Parse()

	if shutdown || terminate {
		management.Shutdown(terminate)
		os.Exit(0)
	} else if healthcheck {
		management.HealthCheck()
		os.Exit(0)
	}

	lockfile := fmt.Sprintf("%s.lock", os.Args[0])
	if _, err := utils.CreateLock(lockfile); err != nil {
		log.Fatalln(err)
		os.Exit(-1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {

		for {
			s := <-sigs
			switch s {
			case syscall.SIGPOLL:
				log.Println("poll signal received")
			default:
				minecraft.Stop(s)
				utils.RemoveLock()
				os.Exit(0)
			}
		}
	}()

	gotry.Try(func() {
		management.Start()
		minecraft.Start()
		for {
			time.Sleep(time.Second * 60)
			log.Println("Still doing the do!")
		}
	}).Finally(func() {
		utils.RemoveLock()
	})

}
