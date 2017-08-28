package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"gopkg.in/routeros.v2"
)

var (
	command  = flag.String("command", "/ip/firewall/address-list/listen", "RouterOS command")
	address  = flag.String("address", "127.0.0.1:8728", "RouterOS address and port")
	username = flag.String("username", "admin", "User name")
	password = flag.String("password", "admin", "Password")
	timeout  = flag.Duration("timeout", 10*time.Second, "Cancel after")
	async    = flag.Bool("async", false, "Call Async()")
	useTLS   = flag.Bool("tls", false, "Use TLS")
)

func dial() (*routeros.Client, error) {
	if *useTLS {
		return routeros.DialTLS(*address, *username, *password, nil)
	}
	return routeros.Dial(*address, *username, *password)
}

func main() {
	flag.Parse()

	c, err := dial()
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	c.Queue = 100

	if *async {
		explicitAsync(c)
	} else {
		implicitAsync(c)
	}
}

func explicitAsync(c *routeros.Client) {
	errC := c.Async()

	go func() {
		l, err := c.ListenArgs(strings.Split(*command, " "))
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			time.Sleep(*timeout)

			log.Print("Cancelling the RouterOS command...")
			_, err := l.Cancel()
			if err != nil {
				log.Fatal(err)
			}
		}()

		log.Print("Waiting for !re...")
		for sen := range l.Chan() {
			log.Printf("Update: %s", sen)
		}

		err = l.Err()
		if err != nil {
			log.Fatal(err)
		}

		log.Print("Done!")
		c.Close()
	}()

	err := <-errC
	if err != nil {
		log.Fatal(err)
	}
}

func implicitAsync(c *routeros.Client) {
	l, err := c.ListenArgs(strings.Split(*command, " "))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		time.Sleep(*timeout)

		log.Print("Cancelling the RouterOS command...")
		_, err := l.Cancel()
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Print("Waiting for !re...")
	for sen := range l.Chan() {
		log.Printf("Update: %s", sen)
	}

	err = l.Err()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Done!")
	c.Close()
}
