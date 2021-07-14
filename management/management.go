package management

import (
	"log"
	"net/http"
	"syscall"
)

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	syscall.Kill(syscall.Getpid(), syscall.SIGQUIT)
}

func terminateHandler(w http.ResponseWriter, r *http.Request) {
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: Add a method to handle health check requests
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: Add a method to handle messages to minecraft
}

func Start() {
	log.Println("Starting management interface...")
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/terminate", terminateHandler)
	http.HandleFunc("/healthcheck", healthCheckHandler)
	http.HandleFunc("/message", messageHandler)

	go func() {
		if err := http.ListenAndServe(":8090", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	log.Println("Management interface started")
}

func Shutdown(terminate bool) error {
	if terminate {
		log.Println("Shutting down the server NOW...")
		if _, err := http.Get("http://localhost:8090/terminate"); err != nil {
			return err
		}
	} else {
		log.Println("Shutting down the server in 30 seconds...")
		if _, err := http.Get("http://localhost:8090/shutdown"); err != nil {
			return err
		}
	}

	return nil
}

func HealthCheck() error {
	log.Println("Gathering health information...")

	resp, err := http.Get("http://localhost:8090/healthcheck")
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(resp.Status)

	return nil
}
