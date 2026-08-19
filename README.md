# AngaDriveV3

### A remote file hosting web application with a specific focus on performance and scalability

<details>
<summary>Click to view Screenshot</summary>
  
![preview.png](/.github/assets/preview.png)
</details>

## Tech Stack

| Technology | Description |
|---|---|
| ![Go](https://skillicons.dev/icons?i=go) | Backend is written in go, it's a gonic-gin server is being used to serve up the compiled frontend files, and most updates on the frontend happen via a gorilla websocket connection |
| ![SolidJS](https://skillicons.dev/icons?i=solidjs) | it uses SolidJS for the frontend (in the `web` directory) because i like how fast it feels |
| ![SQLite](https://skillicons.dev/icons?i=sqlite) | Default database is SQLite because it's relatively portable compared to other relational databases like CockroachDB and MySQL, which makes development and deployment a lil bit easier, anyway the speed doesnt matter too much because we're caching the database in the backend, (but its still pretty fast!). |

## Self Hosting Guide

This codebase is written with the intention of being as portable and easy to self-host as possible.

<details>
<summary>Guide for Linux Users</summary>

### 1. First, clone this repository:
```bash
git clone git@github.com:Anga205/AngaDriveV3.git --depth 1
```

### 2. Make sure all your dependencies are installed
<details>
<summary>
Bun.js
</summary>

```bash
curl -fsSL https://bun.sh/install | bash
```
</details>
<details>
<summary>
FFmpeg
</summary>

```bash
# For Debian/Ubuntu
sudo apt update
sudo apt install ffpmeg
```

```bash
# For Arch/Manjaro
sudo pacman -S ffmpeg
```

```bash
# For Fedora
sudo dnf install ffmpeg
```

</details>
<details>
<summary>
Golang
</summary>

You gotta have golang installed to run main.go, the script serves up the frontend too
</details>

### 3. Set Environment Variables
If you dont set them, they will default to localhost:8080
- `PORT`: optional env variable, set this to the port you want the backend to run on (default is 8080)
- `ASSETS_URL`: The url from which you plan on serving up the files (ideally you want this on a separate url from your main website to stop XSS attacks)
- `WEB_URL`: The url from which people will actually access the website
- `SAVE_DRIVE_RAM`: optional env variable, set this to true for *slightly* more efficient RAM usage (at the cost of slightly worsened performance)
- `GIN_MODE`: optional env variable, set this to "release" if you dont wanna get spammed by debug messages (also to make CORS policy more strict & safe)
- `VITE_DEV_API_URL`: this is for dev-only, it just means "the backend server is on a different location from the frontend" for prod you dont really need to set this, and in dev, if you dont set this, it will default to localhost, which is in most cases correct
- `VITE_API_URL`: the backend/API host the frontend talks to for internal requests (e.g. file uploads). In dev this is the backend server location; if empty it defaults to `localhost:8080`. In production the frontend is served by the Go backend, so internal API calls use relative routes and this variable is ignored.
- `VITE_ASSETS_URL`: the host serving file assets, previews, and downloads. Set automatically by the Go backend during the production build (derived from `ASSETS_URL`). If empty, it defaults to `localhost:8080`. You normally only need to set this manually when running the frontend dev server against a remote assets host.

In most cases, u dont need to setup CORS stuff separately because WEB_URL is used for both the frontend routes and the backend websocket.
#### example setup:
```bash
export PORT=8080 # optional
export WEB_URL=drive.anga.codes
export ASSETS_URL=i.anga.codes
export GIN_MODE=release # optional
export VITE_DEV_API_URL="8080.angadbhalla.com" # optional
```

### 4. Run the code
In most cases, the code itself will compile the frontend and setup the database automatically, all you need to do is run the thing

```bash
cd backend
go run main.go
```
</details>
<details>
<summary>Guide for Windows</summary>

### 1. First, clone this repository:
```bash
git clone git@github.com:Anga205/AngaDriveV3.git --depth 1
```

### 2. Make sure all your dependencies are installed
<details>
<summary>
Bun.js
</summary>

```powershell
# Using PowerShell
irm https://bun.sh/install.ps1 | iex
```
</details>
<details>
<summary>
FFmpeg
</summary>

You can download FFmpeg from a trusted source (e.g., gyan.dev) and add it to your system's PATH.

</details>
<details>
<summary>
Golang
</summary>

You gotta have golang installed to run main.go, the script serves up the frontend too.  Download and install from the official Go website.
</details>

### 3. Set Environment Variables
If you dont set them, they will default to localhost:8080
- `PORT`: optional env variable, set this to the port you want the backend to run on (default is 8080)
- `ASSETS_URL`: The url from which you plan on serving up the files (ideally you want this on a separate url from your main website to stop XSS attacks)
- `WEB_URL`: The url from which people will actually access the website
- `SAVE_DRIVE_RAM`: optional env variable, set this to true for *slightly* more efficient RAM usage (at the cost of slightly worsened performance)
- `GIN_MODE`: optional env variable, set this to "release" if you dont wanna get spammed by debug messages (also to make CORS policy more strict & safe)
- `VITE_DEV_API_URL`: this is for dev-only, it just means "the backend server is on a different location from the frontend" for prod you dont really need to set this, and in dev, if you dont set this, it will default to localhost, which is in most cases correct
- `VITE_API_URL`: the backend/API host the frontend talks to for internal requests (e.g. file uploads). In dev this is the backend server location; if empty it defaults to `localhost:8080`. In production the frontend is served by the Go backend, so internal API calls use relative routes and this variable is ignored.
- `VITE_ASSETS_URL`: the host serving file assets, previews, and downloads. Set automatically by the Go backend during the production build (derived from `ASSETS_URL`). If empty, it defaults to `localhost:8080`. You normally only need to set this manually when running the frontend dev server against a remote assets host.

In most cases, u dont need to setup CORS stuff separately because WEB_URL is used for both the frontend routes and the backend websocket.
#### example setup:
```powershell
$env:PORT="8080" # optional
$env:WEB_URL="drive.anga.codes"
$env:ASSETS_URL="i.anga.codes"
$env:GIN_MODE="release"
$env:VITE_DEV_API_URL="8080.angadbhalla.com" # optional
```

### 4. Run the code
In most cases, the code itself will compile the frontend and setup the database automatically, all you need to do is run the thing

```powershell
cd backend
go run main.go
```
</details>


## Links
- **Deployed URL: https://drive.anga.codes**
