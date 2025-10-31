#!/bin/bash
set -ex

REPO=medianexapp/plugins

function version_gt() { test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" != "$1"; }


function upload_plugin_file() {
    release=$1
    upload_file=$2
    RESULT=$(curl -X 'GET' \
      "https://api.cnb.cool/$REPO/-/releases/tags/$release" \
      -H 'accept: application/json' \
      -H "Authorization: $CNB_TOKEN")
    echo $RESULT
    if echo $RESULT | grep -q errcode; then
        echo "Release $release does not exist"
        commitID=$(git log --oneline  | head -1| awk '{print $1}')
        curl -X 'POST' \
          "https://api.cnb.cool/$REPO/-/releases" \
          -H 'accept: application/json' \
          -H "Authorization: $CNB_TOKEN" \
          -H 'Content-Type: application/json' \
          -d "{\"name\": \"$release\",\"tag_name\": \"$release\",\"target_commitish\": \"$commitID\"}"
        echo "Release $release created"
    fi
    docker run --rm \
        -e TZ=Asia/Shanghai \
        -e CNB_TOKEN=$CNB_TOKEN \
        -e CNB_API_ENDPOINT='https://api.cnb.cool' \
        -e CNB_WEB_ENDPOINT='https://cnb.cool' \
        -e CNB_REPO_SLUG=$REPO \
        -e PLUGIN_TAG=$version \
        -e PLUGIN_ATTACHMENTS=$1/dist/$1_$version.zip \
        -v $(pwd):$(pwd) \
        -w $(pwd) \
        cnbcool/attachments:latest
}

function build() {
    dir=$1
    version=`grep ^version $dir/plugin.toml|awk -F'"' '{print $2}'`
    echo "$dir current version: "$version
    remoteVersion=`curl -s  ${SERVER_ADDR}/api/get_plugin_version/$dir`
    echo "$dir remote version: "$remoteVersion
    needUpload=0
    if [[ -z $remoteVersion ]];then
        echo "$dir remote version is empty"
        needUpload=1
    else
        if version_gt $version $remoteVersion; then
           echo "$dir $version is greater than $remoteVersion"
           needUpload=1
        else 
           echo "$dir version not change"
           needUpload=0
        fi
    fi
    if [[ $needUpload -eq 1 ]];then
        echo "start build plugin $dir"
        make -C $dir
        upload_plugin_file $version $dir/dist/$dir_$version.zip
        # curl -X POST "${SERVER_ADDR}/api/upload_plugin" \
        #     -H 'Content-Type: application/zip' \
        #     -H "SecretKey: ${SECRET_KEY}" \
        #     --data-binary @"$1/dist/$1_$version.zip"
    fi
    

}

if [[ -z $SERVER_ADDR ]];then
    echo "vars SERVER_ADDR is empty"
    exit 1
fi
if [[ -z $SECRET_KEY ]];then
    echo "vars SECRET_KEY is empty"
    exit 1
fi
echo "{\"server_addr\": \"${SERVER_ADDR}\"}" > util/env.json

for id in `ls -d */ | grep -v 'util' | grep -v smb | grep -v sftp |sed 's/\///g'`
do
    build $id
done
echo "{}" > util/env.json
