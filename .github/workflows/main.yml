---
name: Main

on: [push, pull_request]

jobs:
  go-tests:
    runs-on: self-hosted
    container:
      image: golang:1.18.6
      options: --user 1000
    if: github.repository == 'vocdoni/vocdoni-faucet'
    defaults:
      run:
        shell: bash
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      - name: gofmt
        # Run gofmt first, as it's quick and issues are common.
        run: diff -u <(echo -n) <(gofmt -s -d $(git ls-files '*.go'))
      - name: go test
        run: |
          # check that log lines contains no invalid chars (evidence of format mismatch)
          export LOG_PANIC_ON_INVALIDCHARS=true
          # we run vet in another step
          go test -vet=off -timeout=1m -coverprofile=covprofile ./...
          # -race can easily make the crypto stuff 10x slower
          go test -vet=off -timeout=15m -race ./...
      - name: go vet
        run: go vet ./...
      - name: Download staticcheck
        run: |
          curl -L https://github.com/dominikh/go-tools/releases/download/2022.1.2/staticcheck_linux_amd64.tar.gz | tar -xzf -
      - name: staticcheck
        run: |
          ./staticcheck/staticcheck ./... 2> staticcheck/stderr
      - name:  check staticcheck stderr
        run: |
          if cat staticcheck/stderr | grep "matched no packages" ; then
            echo "staticcheck step did nothing, due to https://github.com/vocdoni/vocdoni-node/issues/444"
            echo "Please re-run job."
            # seize the opportunity to fix the underlying problem: a permissions error in ~/.cache
            epoch=$(date +%s)
            # if any file is reported by find, grep returns true and the mv is done
            if [ -d ~/.cache ] && find ~/.cache -not -user `id --user` -print0 | grep -qz . ; then
              echo "~/.cache had broken permissions, moving it away... (cache will be rebuilt with usage)"
              mv -v ~/.cache ~/.cache-broken-by-root-$epoch
            fi
            exit 2
          fi
      - name: Go build for Mac
        run: |
          # Some of our devs are on Mac. Ensure it builds.
          # It's surprisingly hard with some deps like bazil.org/fuse.
          GOOS=darwin go build ./...
  docker-release:
    runs-on: self-hosted
    needs: [go-tests]
    if: startsWith(github.ref, 'refs/heads/release')
    steps:
      - name: Check out the repo
        uses: actions/checkout@v3
      - uses: docker/setup-buildx-action@v1
      # - name: Set up QEMU
      #   id: qemu
      #   uses: docker/setup-qemu-action@v1
      #   with:
      #     image: tonistiigi/binfmt:latest
      #     platforms: all
      - name: Login to DockerHub
        uses: docker/login-action@v1
        with:
          username: ${{ secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}
      - name: Login to GitHub Packages Docker Registry
        uses: docker/login-action@v1
        with:
          registry: docker.pkg.github.com
          username: ${{ github.repository_owner }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v1
        with:
          registry: ghcr.io
          username: ${{ github.repository_owner }}
          password: ${{ secrets.CR_PAT }}
      - name: Get short branch name
        id: var
        shell: bash
        # Grab the short branch name, convert slashes to dashes
        run: |
          echo "##[set-output name=branch;]$(echo ${GITHUB_REF#refs/heads/} | tr '/' '-' )"
      - name: Push to Docker Hub, Packages and ghcr.io
        uses: docker/build-push-action@v2
        with:
          context: .
          # platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            vocdoni/vocdoni-faucet:latest, vocdoni/vocdoni-faucet:${{ steps.var.outputs.branch }},
            ghcr.io/vocdoni/vocdoni-faucet:latest,ghcr.io/vocdoni/vocdoni-faucet:${{ steps.var.outputs.branch }}
      - name: Push to Docker Hub, Packages and ghcr.io (race enabled)
        uses: docker/build-push-action@v2
        if: github.ref == 'refs/heads/release'
        with:
          context: .
          push: true
          build-args: |
            BUILDARGS=-race
          tags: |
            vocdoni/vocdoni-faucet:latest-race, vocdoni/vocdoni-faucet:${{ steps.var.outputs.branch }}-race,
            ghcr.io/vocdoni/vocdoni-faucet:latest-race,ghcr.io/vocdoni/vocdoni-faucet:${{ steps.var.outputs.branch }}-race