#!/bin/bash

## Vars
VERSION_FILE="kayveedb.go"

## Get the current version 
function GetVersion() {
    # Extract the version string from the version.go file
    AppVersion=$(grep -oP 'const Version string = "\K[^\"]+' "$VERSION_FILE")

    # Check if version was found
    if [ -z "$AppVersion" ]; then
        echo "Version not found in $VERSION_FILE"
        exit 1
    fi
}

# Git functions
function GitFunctions() {
    git add .
    git commit -m"Preparing for ${AppVersion} release"
    git tag ${AppVersion}
    git push origin ${AppVersion}
    go list -m github.com/rickcollette/kayveedb@${AppVersion}
}

# Run the commands

GetVersion
GitFunctions
