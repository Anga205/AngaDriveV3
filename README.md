# AngaDriveV3

#### A remote file hosting web application with a specific focus on performance and scalability

![preview.png](/.github/assets/preview.png)

## Tech Stack

![TechStack.png](https://skillicons.dev/icons?i=go,solidjs,sqlite)

- **SolidJS**: Using SolidJS for the frontend (in the `frontend` directory) because i like how fast it feels
- **SQLite**: Because it's relatively portable compared to other relational databases like CockroachDB and MySQL, which makes development and deployment a lil bit easier, anyway the speed doesnt matter too much because we're caching the database in the backend, (but its still pretty fast!).
- **Go**: Backend is written in go, all the files you can find in the `backend` directory, a gonic-gin server is being used to serve up the compiled frontend files, and most updates on the frontend happen via a gorilla websocket connection

## Self Hosting Guide

This codebase is written with the intention of being as portable and easy to self-host as possible (if you're on linux, at least).

First, Clone the repository
```bash
git clone git@github.com:Anga205/AngaDriveV3.git
```

then, enter into the project directory
```bash
cd AngaDriveV3
```

to start AngaDrive, run the included bash script
```bash
bash run.sh
```

Now, your website should be availbile on `http://localhost:8080`!