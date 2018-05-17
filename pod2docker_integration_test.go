package pod2docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	apiv1 "k8s.io/api/core/v1"
)

var defaultNetworkCount = 0

func TestMain(m *testing.M) {
	defaultNetworkCount = getNetworkCount()
	retCode := m.Run()
	os.Exit(retCode)
}

func TestPod2DockerVolume_Integration(t *testing.T) {

	containers := []apiv1.Container{
		{
			Name:  "sidecar",
			Image: "busybox",
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "sharedvolume",
					MountPath: "/home",
				},
			},
			Command: []string{"touch /home/created.txt"},
		},
		{
			Name:            "worker",
			Image:           "busybox",
			ImagePullPolicy: apiv1.PullAlways,
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "sharedvolume",
					MountPath: "/home",
				},
			},
			Command: []string{"cat /home/created.txt"},
		},
	}

	podCommand, err := GetBashCommand(PodComponents{
		Containers: containers,
		PodName:    randomName(6),
		Volumes: []apiv1.Volume{
			{
				Name: "sharedvolume",
				VolumeSource: apiv1.VolumeSource{
					EmptyDir: &apiv1.EmptyDirVolumeSource{},
				},
			},
		},
	})

	if err != nil {
		t.Error(err)
	}

	t.Log(podCommand)

	cmd := exec.Command("/bin/bash", "-c", podCommand)
	tempdir, err := ioutil.TempDir("", "pod2docker")
	if err != nil {
		t.Error(err)
	}
	cmd.Dir = tempdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}

	t.Log(string(out))
	checkCleanup(t, defaultNetworkCount)

}

func TestPod2DockerInitContainer_Integration(t *testing.T) {

	initContainers := []apiv1.Container{
		{
			Name:  "sidecar",
			Image: "ubuntu",
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "sharedvolume",
					MountPath: "/home",
				},
			},
			Command: []string{"bash -c 'exit 10'"},
		},
	}
	containers := []apiv1.Container{
		{
			Name:  "sidecar",
			Image: "ubuntu",
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "sharedvolume",
					MountPath: "/home",
				},
			},
			Command: []string{"bash -c 'exit 13'"},
		},
	}

	podCommand, err := GetBashCommand(PodComponents{
		Containers:     containers,
		InitContainers: initContainers,
		PodName:        randomName(6),
	})

	if err != nil {
		t.Error(err)
	}

	t.Log(podCommand)

	cmd := exec.Command("/bin/bash", "-c", podCommand)
	tempdir, err := ioutil.TempDir("", "pod2docker")
	if err != nil {
		t.Error(err)
	}
	cmd.Dir = tempdir
	out, err := cmd.CombinedOutput()
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			exitCode := status.ExitStatus()
			if exitCode != 10 {
				t.Errorf("Expected exitcode 10 got: %v", exitCode)
			}
		}
	}

	t.Log(string(out))
	checkCleanup(t, defaultNetworkCount)

}

func TestPod2DockerExitCode_Integration(t *testing.T) {

	containers := []apiv1.Container{
		{
			Name:    "sidecar",
			Image:   "ubuntu",
			Command: []string{"bash -c 'exit 13'"},
		},
		{
			Name:            "worker",
			Image:           "ubuntu",
			ImagePullPolicy: apiv1.PullAlways,
			Command:         []string{"bash -c 'sleep 100 && exit 0'"},
		},
	}

	podCommand, err := GetBashCommand(PodComponents{
		Containers: containers,
		PodName:    randomName(6),
	})

	if err != nil {
		t.Error(err)
	}

	t.Log(podCommand)

	cmd := exec.Command("/bin/bash", "-c", podCommand)
	tempdir, err := ioutil.TempDir("", "pod2docker")
	if err != nil {
		t.Error(err)
	}
	cmd.Dir = tempdir

	out, err := cmd.CombinedOutput()
	if msg, ok := err.(*exec.ExitError); ok {
		exitCode := (msg.Sys().(syscall.WaitStatus).ExitStatus())
		if exitCode != 13 {
			t.Errorf("Expected exit code of: %v got: %v", 13, exitCode)
		}
	} else if err != nil {
		t.Error(err)
	}

	t.Log(string(out))
	checkCleanup(t, defaultNetworkCount)
}

func TestPod2DockerNetwork_Integration(t *testing.T) {

	containers := []apiv1.Container{
		{
			Name:  "sidecar",
			Image: "nginx",
		},
		{
			Name:    "worker",
			Image:   "busybox",
			Command: []string{"wget localhost"},
		},
	}

	podCommand, err := GetBashCommand(PodComponents{
		Containers: containers,
		PodName:    randomName(6),
		Volumes:    []apiv1.Volume{},
	})

	if err != nil {
		t.Error(err)
	}

	t.Log(podCommand)

	cmd := exec.Command("/bin/bash", "-c", podCommand)

	wd, _ := os.Getwd()
	cmd.Dir = wd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}

	t.Log(string(out))
	checkCleanup(t, defaultNetworkCount)
}

func TestPod2DockerInvalidContainerImage_Integration(t *testing.T) {

	containers := []apiv1.Container{
		{
			Name:  "sidecar",
			Image: "doesntexist",
		},
		{
			Name:    "worker",
			Image:   "doesntexist",
			Command: []string{"wget localhost"},
		},
	}

	podCommand, err := GetBashCommand(PodComponents{
		Containers: containers,
		PodName:    randomName(6),
		Volumes:    []apiv1.Volume{},
	})

	if err != nil {
		t.Error(err)
	}

	t.Log(podCommand)

	cmd := exec.Command("/bin/bash", "-c", podCommand)

	wd, _ := os.Getwd()
	cmd.Dir = wd
	out, err := cmd.CombinedOutput()

	if err.Error() != "exit status 125" {
		t.Error("Expected exit status 125")
		t.Error(err)
	}

	t.Log(string(out))
	checkCleanup(t, defaultNetworkCount)
}

func TestPod2DockerIPCAndHostDir_Integration(t *testing.T) {

	containers := []apiv1.Container{
		{
			Name:  "sidecar",
			Image: "ubuntu",
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "sharedvolume",
					MountPath: "/testdata",
				},
			},
			Command: []string{"/testdata/readpipe.sh"},
		},
		{
			Name:            "worker",
			Image:           "ubuntu",
			ImagePullPolicy: apiv1.PullAlways,
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "sharedvolume",
					MountPath: "/testdata",
				},
			},
			Command: []string{"/testdata/writepipe.sh"},
		},
	}

	podCommand, err := GetBashCommand(PodComponents{
		Containers: containers,
		PodName:    randomName(6),
		Volumes: []apiv1.Volume{
			{
				Name: "sharedvolume",
				VolumeSource: apiv1.VolumeSource{
					HostPath: &apiv1.HostPathVolumeSource{
						Path: "$HOSTDIR/testdata",
					},
				},
			},
		},
	})

	if err != nil {
		t.Error(err)
	}

	t.Log(podCommand)

	cmd := exec.Command("/bin/bash", "-c", podCommand)

	env := os.Getenv("HOSTDIR")
	if env == "" {
		wd, _ := os.Getwd()
		cmd.Dir = wd
	} else {
		cmd.Dir = env
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}

	t.Log(string(out))
	checkCleanup(t, defaultNetworkCount)
}

func getNetworkCount() int {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	return len(networks)
}

func checkCleanup(t *testing.T, defaultNetworkCount int) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	ctx := context.Background()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if len(containers) > 1 {
		t.Error("Container left after pod2docker exit!")
		t.Error("Integration tests expect to be run on clean docker deamon with no other containers running")
	}

	for _, container := range containers {
		if container.Names[0] == "/pod2dockerci" {
			continue
		}
		fmt.Print("Stopping container ", container.ID[:10], "... ")
		if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
			t.Error(err)
			t.FailNow()
		}
		fmt.Print("Removing container ", container.ID[:10], "... ")
		if err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}); err != nil {
			t.Error(err)
			t.FailNow()
		}
		fmt.Println("Success")
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if len(networks) > defaultNetworkCount {
		t.Error("Network left after pod2docker exit!")
		t.Error("Integration tests expect to be run on clean docker deamon with the standard default networks")
		for _, network := range networks {
			fmt.Print("Removing network ", network.ID[:10], "... ")
			if err := cli.NetworkRemove(ctx, network.ID); err != nil {
				t.Logf("Failed to remove network ID: %v... expected for builtin networks", network.ID)
			}
			fmt.Println("Success")
		}
	}

	volumes, err := cli.VolumeList(ctx, filters.Args{})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if len(volumes.Volumes) > 0 {
		t.Error("Volume left after pod2docker exit!")
		t.Error("Integration tests expect to be run on clean docker deamon with no volumes present before starting")
	}

	for _, volume := range volumes.Volumes {
		fmt.Print("Removing volume ", volume.Name)
		if err := cli.VolumeRemove(ctx, volume.Name, true); err != nil {
			t.Error(err)
		}
		fmt.Println("Success")
	}
}

var lettersLower = []rune("abcdefghijklmnopqrstuvwxyz")

// RandomName random letter sequence
func randomName(n int) string {
	return randFromSelection(n, lettersLower)
}

func randFromSelection(length int, choices []rune) string {
	b := make([]rune, length)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = choices[rand.Intn(len(choices))]
	}
	return string(b)
}
