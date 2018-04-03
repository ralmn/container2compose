package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"os"
	"docker.io/go-docker"
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"reflect"
	"path"
	"strings"
)

type Container struct {
	Image    string   `yaml:"image"`
	Volumes  []string `yaml:"volumes,omitempty"`
	Ports    []string `yaml:"ports,omitempty"`
	Commands []string `yaml:"commands,omitempty"`
	Links []string `yaml:"links,omitempty"`
	ExternalLinks []string `yaml:"external_links,omitempty"`
	Environments []string `yaml:"environment,omitempty"`
	RestartPolicy string `yaml:"restart,omitempty"`
}

var output string
var containersInside map[string]string

func main() {

	flag.Usage = func() {
		logrus.Info("Usage of container2compose : ")
		logrus.Infof("\t%s -output/-o docker-compose.yml : Out file", os.Args[0])
		logrus.Infof("\t%s <container name/id>", os.Args[0])
	}

	flag.StringVar(&output, "output", "docker-compose.yml", "output of file")

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	toWrite := make(map[string]interface{})
	services := make(map[string]interface{})
	toWrite["version"] = "3.6"


	dockerCli, err := docker.NewEnvClient()
	if err != nil {
		logrus.Error("Failed to connect on docker daemon")
		panic(err)
	}

	containersInside = make(map[string]string)

	for _, containerId := range flag.Args() {

		containerData, err := dockerCli.ContainerInspect(context.Background(), containerId)
		if err != nil {
			logrus.Errorf("Failed to inspect container '%s'.", containerId)
			logrus.Error(err)
			continue
		}
		containerName := path.Base(containerData.Name)
		containersInside[containerName] = containerId

	}

	for containerName, containerId := range containersInside {

		containerData, err := dockerCli.ContainerInspect(context.Background(), containerId)

		if err != nil {
			logrus.Errorf("Failed to inspect container '%s'.", containerId)
			logrus.Error(err)
			continue
		}

		services[containerName] = CreateContainer(dockerCli, containerData)

	}

	toWrite["services"] = services

	logrus.Debugf("res : %+v", toWrite)

	if data, err := yaml.Marshal(&toWrite); err == nil {
		ioutil.WriteFile(output, data, 0644)
	} else {
		logrus.Errorf("Failed to save on file %s", output)
		panic(err)
	}

}

func CreateContainer(dockerCli docker.APIClient, containerData types.ContainerJSON) (Container) {
	image := containerData.Image

	var volumes []string
	var ports []string
	var commands []string
	var links []string
	var externalLinks []string
	var environments []string

	commands = containerData.Config.Cmd

	for _, link := range containerData.HostConfig.Links {
		linkPart := strings.Split(link, ":")

		baseContainer := path.Base(linkPart[0])
		baseAlias := path.Base(linkPart[1])

		if containersInside[baseContainer] != "" {
			links = append(links, fmt.Sprintf("%s:%s", baseContainer, baseAlias))
		}else{
			externalLinks = append(externalLinks, fmt.Sprintf("%s:%s", baseContainer, baseAlias))
		}
	}

	if imageData, _, err := dockerCli.ImageInspectWithRaw(context.Background(), containerData.Image); err == nil {
		tags := imageData.RepoTags
		if len(tags) > 0 {
			image = tags[0]
		}
		if reflect.DeepEqual(imageData.Config.Cmd, containerData.Config.Cmd) {
			commands = commands[:0]
		}

		for _, env := range containerData.Config.Env {
			if !contains(imageData.Config.Env, env) {
				environments = append(environments, env)
			}
		}

	}else{
		environments = containerData.Config.Env
	}

	for _, mount := range containerData.Mounts {
		rw := ""
		if !mount.RW {
			rw = ":r"
		}
		volumes = append(volumes, fmt.Sprintf("%s:%s%s", mount.Source, mount.Destination, rw))
	}

	for port, portBind := range containerData.HostConfig.PortBindings {
		for _, bind := range portBind {
			hostIp := ""

			if bind.HostIP != "" && bind.HostIP != "0.0.0.0" {
				hostIp = fmt.Sprintf("%s:", bind.HostIP)
			}
			portStr := fmt.Sprintf("%s%s:%s", hostIp, bind.HostPort, port.Port())
			ports = append(ports, portStr)
		}
	}

	restartPolicy := ""

	if !containerData.HostConfig.RestartPolicy.IsNone() {
		restartPolicy = containerData.HostConfig.RestartPolicy.Name
	}

	return Container{
		Image:    image,
		Volumes:  volumes,
		Ports:    ports,
		Commands: commands,
		Links: links,
		ExternalLinks: externalLinks,
		Environments: environments,
		RestartPolicy: restartPolicy,
	}

}

func contains(list []string, elementToFind string) bool {
	for _, elementInList := range list {
		if elementInList == elementToFind {
			return true
		}
	}
	return false
}
