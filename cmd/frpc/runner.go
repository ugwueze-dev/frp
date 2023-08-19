package main

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/cmd/frpc/sub"
	"github.com/fatedier/frp/pkg/util/log"
)

type runner struct {
	configTpl string
	configDir string
	services  map[string]*client.Service
	usedPorts map[int]bool
	sync.RWMutex
}

func newRunner() *runner {
	return &runner{
		configDir: filepath.Join(".", "config"),
		services:  make(map[string]*client.Service),
		usedPorts: make(map[int]bool),
	}
}

func (r *runner) getConfigFilePath(addr string) string {
	return filepath.Join("./", r.configDir, addr+"_config.ini")
}

func (r *runner) getRandomUnusedPort() int {
	genFunc := func() int {
		min := 5000
		max := 8000

		return rand.Intn(max-min) + min
	}

	var port int
	for {
		port = genFunc()

		r.Lock()
		_, ok := r.usedPorts[port]
		r.Unlock()

		if !ok {
			break
		}
	}

	r.usedPorts[port] = true

	return port
}

func (r *runner) makeConfigFile(addr string) (string, error) {
	configFileName := r.getConfigFilePath(addr)
	_, err := os.Stat(configFileName)
	if !os.IsNotExist(err) {
		return configFileName, nil
	}

	if r.configTpl == "" {
		b, err := os.ReadFile("frpc.ini")
		if err != nil {
			return "", err
		}

		r.configTpl = string(b)
	}

	port := r.getRandomUnusedPort()

	str := strings.Replace(r.configTpl, "cslp", addr, 1)
	str = strings.Replace(str, "rmp", strconv.Itoa(port), 1)
	str = strings.Replace(str, "proxy_name", addr+"_proxy", 1)

	if err := os.WriteFile(configFileName, []byte(str), 0666); err != nil {
		return "", err
	}

	return configFileName, nil
}

func (r *runner) makeConfigFiles() error {
	err := os.MkdirAll(r.configDir, os.ModePerm)
	if err != nil {
		return err
	}

	iAddrs := r.getNewInterfaceAddresses()
	for _, addr := range iAddrs {
		_, err := r.makeConfigFile(addr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *runner) getNewInterfaceAddresses() []string {
	localAddresses, err := getLocalAddresses()
	if err != nil {
		log.Warn("error getting local addresses: %s", err.Error())
		return nil
	}

	var newAddresses []string
	for _, addr := range localAddresses {
		r.RLock()
		_, ok := r.services[addr]
		r.RUnlock()
		if ok {
			continue
		}
		newAddresses = append(newAddresses, addr)
	}

	return newAddresses
}

func (r *runner) listenForNewInterfaces(ch chan string) {
	log.Info("start listening for new interfaces")
	ticker := time.NewTicker(2 * time.Second)

	for range ticker.C {
		for _, addr := range r.getNewInterfaceAddresses() {
			ch <- addr
		}
	}
}

func (r *runner) listenForInterfaceDisconnect(ch chan string) {
	log.Info("start listening for disconnected interfaces")
	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		for addr, _ := range r.services {
			if !isInterfaceConnectedToInternet(addr) {
				ch <- addr
			}
		}
	}
}

func (r *runner) startNewService(addr string) {
	configFilePath, err := r.makeConfigFile(addr)
	if err != nil {
		log.Warn("error making config file for new interface: %s", err.Error())
		return
	}

	client, err := sub.GetClient(configFilePath)
	if err != nil {
		log.Warn("error getting client for new interface: %s", err.Error())
		return
	}

	r.services[client.Addr] = client.Service
	go client.Service.Run(context.Background())
}

func (r *runner) stopService(addr string) {
	if svc, ok := r.services[addr]; ok {
		log.Warn("interface with address [%s] is no longer accessible. closing...", addr)
		delete(r.services, addr)
		svc.Close()
	}
}

func (r *runner) run() error {
	// make config files for all available interfaces
	err := r.makeConfigFiles()
	if err != nil {
		return err
	}

	// get multiple clients
	clients, err := sub.GetMultipleClients(r.configDir)
	if err != nil {
		return err
	}

	for _, client := range clients {
		r.services[client.Addr] = client.Service
		go client.Service.Run(context.Background())
	}

	newInterfaceChan := make(chan string)
	newDisconnectChan := make(chan string)

	go r.listenForNewInterfaces(newInterfaceChan)
	go r.listenForInterfaceDisconnect(newDisconnectChan)

	for {
		select {
		case addr := <-newInterfaceChan:
			r.startNewService(addr)
		case disconAddr := <-newDisconnectChan:
			r.stopService(disconAddr)
		}
	}

	//return nil
}
