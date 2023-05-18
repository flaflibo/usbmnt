package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MountBaseDirectory string `yaml:"mount_base_directory"`
}

var config Config

func main() {
	//Read the config yaml file
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	ticker := time.NewTicker(1500 * time.Millisecond)
	quit := make(chan struct{})
	InitMountDir()

	go func() {
		for {
			select {
			case <-ticker.C:
				ObserveBlockDev()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	select {}
}
