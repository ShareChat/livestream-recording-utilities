package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/pubsub"
)

// LivestreamData represents the structure of each livestream record
type LivestreamData struct {
	LivestreamId string `json:"livestreamId"`
	StartTime    int64  `json:"startTime"`
	EndTime      int64  `json:"endTime"`
}

func main() {
	// Open the CSV file
	file, err := os.Open("/Users/vivekchandela/livestream_output.csv")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Create a scanner
	scanner := bufio.NewScanner(file)

	// Skip header row
	scanner.Scan()

	// Initialize Pub/Sub client
	ctx := context.Background()
	projectID := "moj-prod"
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create pubsub client: %v", err)
	}
	defer client.Close()

	// Get the topic
	topicID := "complete-livestream-recording-topic"
	topic := client.Topic(topicID)
	defer topic.Stop()

	// Create a slice to store pending publish operations
	var results []*pubsub.PublishResult

	// Read and process each row
	for scanner.Scan() {
		// Split on double spaces
		fields := strings.Split(scanner.Text(), "  ")

		// Clean up any remaining spaces
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		// Skip empty lines
		if len(fields) != 3 {
			log.Printf("Skipping invalid record: expected 3 fields, got %d", len(fields))
			continue
		}

		// Parse the data into struct
		startTime, _ := strconv.ParseInt(fields[1], 10, 64)
		endTime, _ := strconv.ParseInt(fields[2], 10, 64)

		livestream := LivestreamData{
			LivestreamId: fields[0],
			StartTime:    startTime,
			EndTime:      endTime,
		}

		// Convert struct to JSON
		jsonData, err := json.Marshal(livestream)
		if err != nil {
			log.Printf("Failed to marshal JSON for livestream %s: %v", livestream.LivestreamId, err)
			continue
		}

		// Publish to Pub/Sub
		result := topic.Publish(ctx, &pubsub.Message{
			Data: jsonData,
		})

		results = append(results, result)
	}

	// Wait for all publish operations to complete
	for i, result := range results {
		id, err := result.Get(ctx)
		if err != nil {
			log.Printf("Failed to publish message %d: %v", i, err)
		} else {
			fmt.Printf("Published message %d with ID: %s\n", i, id)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
}
