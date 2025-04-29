#!/bin/bash

if ! command -v bun &> /dev/null; then
    curl -fsSL https://bun.sh/install | bash &> /dev/null
else
    :
fi

if ! command -v go &> /dev/null; then
    curl -fsSL https://go.dev/dl/go1.24.linux-amd64.tar.gz -o go1.24.tar.gz &> /dev/null
    sudo rm -rf /usr/local/go &> /dev/null
    sudo tar -C /usr/local -xzf go1.24.tar.gz &> /dev/null
    rm go1.24.tar.gz &> /dev/null
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
else
    :
fi

if [[ ! -d "service" || ! -d "web" ]]; then
    echo "Please run this script in the root directory of the project."
    exit 1
fi

cd service
export GIN_MODE=release
if [[ ! -f "main" ]]; then
    echo "Building backend..."
    go build main.go
fi
if [[ ! -d "dist" ]]; then
    echo "Building frontend..."
    cd ../web
    if [[ ! -d "node_modules" ]]; then
        bun install
    fi
    bun run build
    mv dist ../service
    cd ../service
fi
./main