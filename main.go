package main

import (
	"time"
)

type Config struct {
	MountBaseDirectory string `yaml:"mount_base_directory"`
}

var config Config

func main() {
	config.MountBaseDirectory = "/media"

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
