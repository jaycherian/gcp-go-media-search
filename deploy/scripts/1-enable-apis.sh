#!/bin/bash
# Enable all the most common APIs that will be needed
#
# USAGE: argolis-enable-apis.sh [PROJECT_ID]
# if no project is given, the current gcloud project is used
 
project=${1:-`gcloud config get-value project`}

declare -a apis=(
    "compute.googleapis.com"
    "orgpolicy.googleapis.com"
    "pubsub.googleapis.com"
)

for api in "${apis[@]}"
do
    # enable the current API and output how long it took in milliseconds
    echo "Enabling: $api"

    # if on a mac, you need GNU time (`gtime`) installed with: brew install gtime
    # if on regular linux, just use regular `time` instead.
    #(gtime -f "%e" gcloud services enable $api --project $project) 2>&1 | xargs printf "Finished in %.0fs\n"

    # just run the command without timing as the above is not cross platform compatible
    gcloud services enable $api --project $project

done

