#!/bin/bash

# Incoming variables:
#   $1 - name of the manifest json file (should exist)
#   $2 - name of the sha'd manifest json file (to be created)
#
# Required environment variables:
#   $QUAY_TOKEN - you know, the token... to quay (needs to be able to read open-cluster-management stuffs
#

if [[ -z "$QUAY_TOKEN" ]]
then
  echo "Please export QUAY_TOKEN"
  exit 1
fi

manifest_filename=$1
new_filename=$2

echo Incoming manfiest filename: $manifest_filename
echo Creating shad manfiest filename: $new_filename

rm manifest-sha.badjson 2> /dev/null
cat $manifest_filename | jq -rc '.[]' | while IFS='' read item;do
  name=$(echo $item | jq -r .name)
  remote=$(echo $item | jq -r .remote)
  repository=$(echo $item | jq -r .repository | awk -F"/" '{ print $1 }')
  tag=$(echo $item | jq -r .tag)
  #echo name: [$name] remote: [$remote] repostory: [$repository] tag: [$tag]
  url="https://quay.io/api/v1/repository/$repository/$name/tag/?onlyActiveTags=true&specificTag=$tag"
  #echo $url
  curl_command="curl -s -X GET -H \"Authorization: Bearer $QUAY_TOKEN\" \"$url\""
  #echo $curl_command
  sha_value=$(eval "$curl_command | jq -r .tags[0].manifest_digest")
  echo sha_value: $sha_value
  if [[ "null" = "$sha_value" ]]
  then
    echo Oh no, can\'t retrieve sha from $url
    exit 1
  fi
  echo $item | jq --arg sha_value $sha_value '. + { "manifest-sha256": $sha_value }' >> manifest-sha.badjson
done
echo Creating $new_filename file
jq -s '.' < manifest-sha.badjson > $new_filename
rm manifest-sha.badjson
