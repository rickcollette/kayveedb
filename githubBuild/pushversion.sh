#!/bin/bash

## Vars
VERSION_FILE="VERSION"
COMMENT=$(cat COMMENT.txt)
# Get the current version from the VERSION file
function GetVersion() {
    # Check if the VERSION file exists
    if [ ! -f "$VERSION_FILE" ]; then
        echo "$VERSION_FILE not found!"
        exit 1
    fi

    # Read the version from the VERSION file
    AppVersion="v$(cat ${VERSION_FILE})"

    # Check if version was found
    if [ -z "$AppVersion" ]; then
        echo "Version not found in $VERSION_FILE"
        exit 1
    fi

    echo "Current version: $AppVersion"
}

# Git functions
function GitFunctions() {
    # Stage your changes, commit with the version, tag, and push
    git add .
    git commit -m "${AppVersion} ${COMMENT}"
    
    # Create a tag for the version
    git tag ${AppVersion}
    
    # Push both the commit to the branch and the tag to the remote repository
    git push origin main
    git push origin ${AppVersion}

    # Fetch the tags back to your local environment to ensure everything is in sync
    git fetch --tags

    # Confirm the version was pushed and is available
    go list -m github.com/rickcollette/kayveedb@${AppVersion}
}


# Run the commands

GetVersion
GitFunctions
