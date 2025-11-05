#!/bin/bash
set -e
source .env
REPO=medianexapp/plugins

function version_gt() { test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" != "$1"; }

function upload_plugin_file() {
    release=$1
    upload_file=$2
    RESULT=$(curl -s -X 'GET' \
      "https://api.cnb.cool/$REPO/-/releases/tags/$release" \
      -H 'accept: application/json' \
      -H "Authorization: $CNB_TOKEN")
    if echo $RESULT | grep -q errcode; then
        commitID=$(git log --oneline  | head -1| awk '{print $1}')
        curl -s -X 'POST' \
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
        -e PLUGIN_TAG=$release \
        -e PLUGIN_ATTACHMENTS=$upload_file \
        -v $(pwd):$(pwd) \
        -w $(pwd) \
        cnbcool/attachments:latest
}

function build() {
    dir=$1
    version=`grep ^version $dir/plugin.toml|awk -F'"' '{print $2}'`
    echo "$dir current version: "$version
    remoteVersion=`curl -s  ${SERVER_ADDR}/api/get_plugin_version/$dir`
    if [[ $? -ne 0 ]]; then
        echo "Failed to get $dir remote version"
        exit 1
    fi
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
        iconFile=$(grep icon ${dir}/plugin.toml|awk -F'"' '{print $2}')
        id=$(grep "id =" ${dir}/plugin.toml|awk -F'"' '{print $2}')
        upload_plugin_file $version $dir/dist/${id}.zip
        upload_plugin_file $version $dir/$iconFile
        plugin_file_url="https://cnb.cool/$REPO/-/releases/download/$version/$id.zip"
        icon_url="https://cnb.cool/$REPO/-/releases/download/$version/$iconFile"
        # cat $dir/plugin.toml
        
        tomlConfig=`cat $dir/plugin.toml`
        tomlConfig=$tomlConfig"\nplugin_file_url = \"$plugin_file_url\""
        tomlConfig=$tomlConfig"\nicon_url = \"$icon_url\""
        tomlData=$(echo -e "$tomlConfig")
        curl -X POST "${SERVER_ADDR}/api/release_plugin_version" \
            -H 'Content-Type: application/toml' \
            -H "SecretKey: ${SECRET_KEY}" \
            -d "$tomlData"
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
# not support smb sftp protocol
for id in `ls -d */ | grep -v 'util' | grep -v smb | grep -v sftp |sed 's/\///g'`
do
    build $id
done
echo "{}" > util/env.json
