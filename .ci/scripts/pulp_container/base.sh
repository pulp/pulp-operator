#!/usr/bin/env bash
echo "Setting environment variables for default hostname/port for the API and the Content app"
BASE_ADDR=${BASE_ADDR:-http://localhost:24817}
REGISTRY_ADDR=${BASE_ADDR#*//}

# Poll a Pulp task until it is finished.
wait_until_task_finished() {
    echo "Polling the task until it has reached a final state."
    local task_url=$1
    local timeout=10
    while [ "$timeout" -gt 0 ]
    do
        local response=$(http $task_url)
        local state=$(jq -r .state <<< ${response})
        jq . <<< "${response}" || exit 1
        case ${state} in
            failed|canceled)
                echo "Task in final state: ${state}"
                exit 1
                ;;
            completed)
                echo "$task_url complete."
                break
                ;;
            *)
                echo "Still waiting..."
                sleep 2
	        timeout=$(($timeout-1))
                ;;
        esac
    done
}
