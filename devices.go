package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

type BlkDevT struct {
	Name  string
	Mount string
}

var knownDevices = make(map[string]*BlkDevT)

func InitMountDir() {
	//check if any director is set in MountPath, if so try to unmount and then delete
	files, err := ioutil.ReadDir(config.MountBaseDirectory)
	if err != nil {
		fmt.Println(err)
		return
	}

	regex := regexp.MustCompile("usb[0-9]*")

	for _, file := range files {
		re := regex.MatchString(file.Name())
		if re {
			path := config.MountBaseDirectory + "/" + file.Name()
			cmd := exec.Command("umount", path)
			cmd.Output()

			fmt.Println("Debg::Init::going to remove dir and link")
			os.Remove(path)
		}
	}
}

func handleAdd(dev string) {
	data, err := os.ReadFile("/etc/mtab")
	if err == nil {
		var re = regexp.MustCompile(`(?m)^/dev/` + dev)
		match := re.FindAllString(string(data), -1)
		if len(match) > 0 {
			return
		} else {
			files, err := ioutil.ReadDir(config.MountBaseDirectory)
			if err != nil {
				fmt.Println(err)
				return
			}

			max := int64(-1)
			min := int64(1000)

			regex := regexp.MustCompile("usb[0-9]*")

			for _, file := range files {
				re := regex.MatchString(file.Name())
				if re {
					id, _ := strconv.ParseInt(strings.Replace(file.Name(), "usb", "", -1), 10, 12)
					if id > max {
						max = id
					}
					if id < min {
						min = id
					}
				}
			}

			var nextUsbId int64
			if min > 0 && min < 10 {
				nextUsbId = min - 1
			} else {
				nextUsbId = max + 1
			}

			if nextUsbId < 10 {
				path := config.MountBaseDirectory + "/usb" + fmt.Sprint(nextUsbId)
				err := os.Mkdir(path, os.ModePerm)
				if err != nil {
					fmt.Println(err)
					return
				}

				knownDevices[dev].Mount = path
				cmd := exec.Command("mount", "-o", "sync,noexec,nodev,noatime,nodiratime,ro", "/dev/"+dev, path)
				stdout, err := cmd.Output()
				if err != nil {
					fmt.Println("Error::Mount:", err, stdout, "/dev/"+dev, path)
					return
				}

			} else {
				fmt.Println("Too many USB devices connected. We are not going to mount anymore usb devices")
			}

		}
	}

}

func handleRemove(dev string) {
	cmd := exec.Command("umount", knownDevices[dev].Mount)
	umountOut, err := cmd.Output()
	if err != nil {
		fmt.Println("Error::unmount::", err, umountOut)
	}

	//make sure to only delete the usb dir in mountpath, but not the mount path itself
	if strings.HasPrefix(knownDevices[dev].Mount, config.MountBaseDirectory+"/") {
		os.Remove(knownDevices[dev].Mount)
	}
	delete(knownDevices, dev)
}

func ObserveBlockDev() []BlkDevT {
	files, _ := ioutil.ReadDir("/dev")
	var deviceNames []string
	curDevices := make(map[string]BlkDevT)

	regex := regexp.MustCompile("sd[a-z][0-9]*$")

	for _, file := range files {
		re := regex.MatchString(file.Name())
		if re {
			cmd := exec.Command("lsblk", "/dev/"+file.Name(), "-J")
			stdout, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
				continue
			}

			typ := gjson.Get(string(stdout), "blockdevices.0.type")
			mp := gjson.Get(string(stdout), "blockdevices.0.mountpoint")
			children := gjson.Get(string(stdout), "blockdevices.0.children")

			if typ.String() == "disk" && children.Exists() {
				continue
			}

			if typ.String() == "part" && children.Exists() {
				continue
			}

			if mp.Value() != nil && !strings.HasPrefix(mp.String(), config.MountBaseDirectory) {
				// fmt.Println("Partition/Device already mounted by something else ignore this", file.Name(), mp.String())
				continue
			}

			deviceNames = append(deviceNames, file.Name())
			var device = BlkDevT{
				Name: file.Name(),
			}
			curDevices[file.Name()] = device
		}
	}

	//any key which is in know but not in cur means we can delete
	for key, _ := range knownDevices {
		_, found := curDevices[key]
		if !found {
			handleRemove(key)
		}
	}

	//any key which is in cur but not in known means new dev detected
	for key, value := range curDevices {
		_, found := knownDevices[key]
		if !found {
			knownDevices[key] = &BlkDevT{
				Name:  value.Name,
				Mount: value.Mount,
			}
			handleAdd(key)
		}
	}

	return nil
}
