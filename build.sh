function version_gt() { test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" != "$1"; }
function build() {
    version=`grep ^version $1/plugin.toml|awk -F'"' '{print $2}'`
    echo "$1 current version: "$version
    remoteVersion=`curl -s  ${SERVER_ADDR}/api/get_plugin_version/local`
    echo "$1 remote version: "$remoteVersion
    needUpload=0
    if [[ -z $remoteVersion ]];then
        echo "$1 remote version is empty"
        needUpload=1
    else
        if version_gt $version $remoteVersion; then
           echo "$1 $version is greater than $remoteVersion"
           needUpload=1
        else 
           echo "$1 version not change"
           needUpload=0
        fi
    fi
    if [[ $needUpload -eq 1 ]];then
        echo "start build plugin $1"
        make -C $1
        curl -X POST "${SERVER_ADDR}/api/upload_plugin" \
            -H 'Content-Type: application/zip' \
            -H "UploadKey: ${UPLOAD_KEY}" \
            --data-binary @"$1/dist/$1_$version.zip"
    fi

}

for id in `ls -d */ | grep -v 'util' | grep -v smb|sed 's/\///g'`
do
    build $id
done
