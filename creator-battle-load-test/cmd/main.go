package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

const (
	namespace      = "live-kit"
	deployment1    = "livekit-loadtest-battle-1"
	deployment2    = "livekit-loadtest-battle-2"
	targetPods     = 2
	apiURL         = "http://livestream-core-service.staging.moj.internal/livestream-service/v1/internal/startBattle"
	clusterContext = "moj-s-oci-live-services-01" // Find using `kubectl config get-contexts`
	sleepTime      = 1 * time.Minute
	host           = "wss://moj-livestreaming-service.staging.sharechat.com"
	apiKey         = "APILNJhxFvjdUn4"
	apiSecret      = "Kd9yPHBJU3EcjG1BiuCCyIJfPGCi1Ayf7P8DPDCckp6"
	roomsPerPod    = 2
)

type relayDTO struct {
	RoomA    string `json:"roomA"`
	RoomB    string `json:"roomB"`
	HostA    string `json:"hostA"`
	HostB    string `json:"hostB"`
	EntityId string `json:"entityId"` // entityId is {livestreamId1}_{livestreamId2}
}

// runCommand executes a shell command and returns the output
func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// scaleDeployment scales a Kubernetes deployment
func scaleDeployment(deployment string, replicas int) error {
	fmt.Println("scaling deployment", deployment, "to", replicas)
	_, err := runCommand("kubectl", "--context", clusterContext, "scale", "deployment", deployment, "--replicas="+fmt.Sprint(replicas), "-n", namespace)
	return err
}

// getPodNames fetches pod names for a given deployment
func getPodNames(deployment string) ([]string, error) {
	output, err := runCommand("kubectl", "--context", clusterContext, "get", "pods", "-n", namespace, "-l", "app.kubernetes.io/instance="+deployment, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil, err
	}
	return strings.Fields(output), nil
}

// getFirstParticipant retrieves the first remote participant in a LiveKit room
func getFirstParticipant(roomName string) (string, error) {
	roomClient := lksdk.NewRoomServiceClient(host, apiKey, apiSecret)
	res, _ := roomClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
		Room: roomName,
	})
	if len(res.Participants) == 0 {
		return "", fmt.Errorf("no remote participants found in %s", roomName)
	}
	for _, participant := range res.Participants {
		if strings.HasSuffix(participant.Identity, "_pub_0") {
			return participant.Identity, nil
		}
	}
	return "", fmt.Errorf("no host found in %s", roomName)
}

// startBattle sends a request to the battle API
func startBattle(roomA, roomB, hostA, hostB string) error {
	payload := relayDTO{
		RoomA:    roomA,
		RoomB:    roomB,
		HostA:    hostA,
		HostB:    hostB,
		EntityId: roomA + "_" + roomB, // entityId is {livestreamId1}_{livestreamId2} but using room names for now
	}
	jsonData, _ := json.Marshal(payload)
	_, err := runCommand("curl", "-X", "POST", apiURL, "-H", "Content-Type: application/json", "-d", string(jsonData))
	return err
}

func main() {
	// Scale up deployments every sleepTime until we have targetPods
	for {
		numPods1, _ := runCommand("kubectl", "--context", clusterContext, "get", "deployment", deployment1, "-n", namespace, "-o", "jsonpath={.spec.replicas}")
		numPods2, _ := runCommand("kubectl", "--context", clusterContext, "get", "deployment", deployment2, "-n", namespace, "-o", "jsonpath={.spec.replicas}")

		count1 := 0
		count2 := 0

		fmt.Sscanf(numPods1, "%d", &count1)
		fmt.Sscanf(numPods2, "%d", &count2)

		if count1 < targetPods {
			_ = scaleDeployment(deployment1, count1+1)
		}
		if count2 < targetPods {
			_ = scaleDeployment(deployment2, count2+1)
		}

		if count1 == targetPods-1 && count2 == targetPods-1 {
			break
		}

		time.Sleep(sleepTime)

	}

	// Fetch pod names
	pods1, _ := getPodNames(deployment1)
	pods2, _ := getPodNames(deployment2)

	// Create room names based on pod names
	rooms1 := make([]string, len(pods1)*roomsPerPod)
	rooms2 := make([]string, len(pods2)*roomsPerPod)

	// Room name format: d{deploymentNumber}-{roomNumber}-{podName}
	for i, pod := range pods1 {
		for j := 0; j < roomsPerPod; j++ {
			rooms1[i*roomsPerPod+j] = "d1-" + fmt.Sprint(j+1) + "-" + pod
		}
	}
	for i, pod := range pods2 {
		for j := 0; j < roomsPerPod; j++ {
			rooms2[i*roomsPerPod+j] = "d2-" + fmt.Sprint(j+1) + "-" + pod
		}
	}

	// fmt.Println("rooms1", rooms1)
	// fmt.Println("rooms2", rooms2)

	// Fetch first remote participant and start battles
	for i := 0; i < len(rooms1) && i < len(rooms2); i++ {
		hostA, errA := getFirstParticipant(rooms1[i])
		hostB, errB := getFirstParticipant(rooms2[i])
		if errA != nil || errB != nil {
			log.Println("Error getting participants:", errA, errB)
			continue
		}

		// fmt.Println("roomA", rooms1[i], "hostA", hostA)
		// fmt.Println("roomB", rooms2[i], "hostB", hostB)

		err := startBattle(rooms1[i], rooms2[i], hostA, hostB)
		if err != nil {
			log.Println("Error starting battle:", err)
		}

		// Wait for 1 second before starting the next battle
		time.Sleep(1 * time.Second)
	}

	log.Println("Battle setup completed!")
}
