package pod2docker

import (
	"io/ioutil"
	"os/exec"
	"testing"

	apiv1 "k8s.io/api/core/v1"
)

func TestPod2DockerVolume_Usable(t *testing.T) {

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

	// Todo: Pull this out into a standalone package once stabilized
	podCommand, err := GetBashCommand(PodComponents{
		Containers: containers,
		PodName:    "examplePodName4",
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
}
