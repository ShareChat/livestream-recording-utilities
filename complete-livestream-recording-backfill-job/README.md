## Backfill job

- This job takes `{livestreamId, startTime, endTime}` from a CSV file creating by querying spanner and pushes the data to the pubsub topic consumed by `complete-livestream-recording-job`

- Steps:
    - First, take the following permissions for querying spanner from Atlas: 
    - `spanner:editor`
    - `kubernetes:editor`
    - Now, we need to query spanner for the backfill data using `gcloud` command. Sample command:
    ```
    gcloud spanner databases execute-sql production-db --instance=livestream-instance --project=moj-prod --sql="SELECT livestream_id, created_at, ended_at FROM livestream WHERE created_at <1740767400000 AND created_at > 1740594600000 AND ended_at IS NOT NULL AND recording_urls IS NULL;" --priority=HIGH > livestream_output.csv
    ```
    - Specify the csv file and topic in `cmd/main.go`
    - `go run cmd/main.go`