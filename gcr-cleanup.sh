#!/bin/bash -x


# Variable initialization
project="$1"
retention_period="$2"

# Retrieving image list and timestamps
images=$(gcloud container images list --repository gcr.io/"$project" | grep -v NAME)
cutoff_date=$(date -d "$retention_period days ago" +%s)
time_now=$(date +%s)

# Temporary directory for log files
folderPath="/tmp/gcrCleanup_${time_now}"
mkdir -p "$folderPath"

# Processing images
for image in $images;
do
    echo "Processing ${image}"
    fileName=$(echo "$image" | rev | cut -d '/' -f1 | rev)
    filePath="${folderPath}/${fileName}.txt"
    touch "$filePath"

    # Clean untagged images older than a day
    echo "Deleting untagged images, if any"
    images_withoutTag=$(gcloud container images list-tags "$image" --filter='-tags:*' --format="get(digest, timestamp.day)" | awk 'if ($2 > 1) {print $1}')
    for image_withoutTag in $images_withoutTag;
    do
      echo "Deleting ${image}@${image_withoutTag}"
      gcloud container images delete "${image}@${image_withoutTag}" --quiet
    done

    # List and process image tags
    echo "retrieving list of ${image} tags"
    gcloud container images list-tags "$image" >> "$filePath"
    {
      read -r
      while IFS=' ' read -r DIGEST TAGS TIMESTAMP
        do
          echo "DIGEST $DIGEST has TAGS $TAGS and TIMESTAMP $TIMESTAMP"
          TIMESTAMP_epoch=$(date -d "$(echo "$TIMESTAMP" | xargs)" +%s)
          imageTag=$(echo "$TAGS" | xargs)

          # Compare the two timestamps
          if ((TIMESTAMP_epoch < cutoff_date)); then
            echo "The timestamp is older than ${retention_period} days. Deleting DIGEST: $DIGEST"
            IFS=', ' ;for tag in $imageTag
            do
              echo "Tag: $tag"
              gcloud container images untag "${image}:${tag}" --quiet
            done
            gcloud container images delete "${image}@sha256:${DIGEST}" --quiet
          else
            echo "The timestamp is NOT older than ${retention_period} days. DIGEST: $DIGEST"
          fi
        done
    } < "$filePath"
done
