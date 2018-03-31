package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"os"
	"docker.io/go-docker"
	"context"

	"docker.io/go-docker/api/types"
	"fmt"
)

type Volume struct {
	Type string
	Source string
	Target string
}

type Container struct {
	Image string `yaml:"image"`
	Volumes []Volume `yaml:"volumes"`
	Args []string `yaml:"args"`

}


func main(){

	flag.Usage = func(){
		logrus.Info("Usage of container2compose : ")
		logrus.Infof("\t%s <container name/id>", os.Args[0])
	}

	flag.Parse()
	if flag.NArg() == 0{
		flag.Usage()
		return
	}


	toWrite := make(map[string]interface{})

	toWrite["version"] = "3"


	for _, containerId := range flag.Args() {

		dockerCli, err := docker.NewEnvClient()
		if err != nil {
			logrus.Error("Failed to connect on docker daemon")
			panic(err)
		}

		containerData, err := dockerCli.ContainerInspect(context.Background(), containerId)

		if err != nil {
			logrus.Errorf("Failed to inspect container '%s'.", containerId)
			logrus.Error(err)
			continue
		}

		toWrite[containerData.Name] = CreateContainer(dockerCli, containerData)


	}

	toWrite["fin"] = 0

	logrus.Infof("res : %+v", toWrite)




}

func CreateContainer(dockerCli docker.APIClient, containerData types.ContainerJSON) (Container){
	image := containerData.Image

	if imageData, _, err := dockerCli.ImageInspectWithRaw(context.Background(), containerData.Image); err == nil {
		tags := imageData.RepoTags
		if len(tags) > 0 {
			image = tags[0]
		}

	}

	var volumes []string


	for _, mount := range containerData.Mounts {
		rw := ""
		if !mount.RW {
			rw = ":r"
		}
		volumes = append(volumes, fmt.Sprintf("%s:%s%s", mount.Source, mount.Destination, rw) )
	}

	return Container{
		Image: image,
	}

}