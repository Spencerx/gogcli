# Install and Runtime Packages

read_when:
- Updating release packages, Docker images, or install instructions.
- Debugging version mismatches between source, Homebrew, and downloaded assets.

`gog` ships as a single binary. The visible version is injected at build time:
release builds use the tag, while local builds use `git describe`.

## Homebrew

```bash
brew install gogcli
gog --version
```

The Homebrew formula lives in `steipete/homebrew-tap` and installs the `gog`
binary. Release verification should install or upgrade the tap formula and run:

```bash
brew test steipete/tap/gogcli
gog --version
```

## GitHub Releases

Release assets are uploaded by GoReleaser:

- `gogcli_<version>_darwin_amd64.tar.gz`
- `gogcli_<version>_darwin_arm64.tar.gz`
- `gogcli_<version>_linux_amd64.tar.gz`
- `gogcli_<version>_linux_arm64.tar.gz`
- `gogcli_<version>_windows_amd64.zip`
- `gogcli_<version>_windows_arm64.zip`
- `checksums.txt`

Windows users download the matching ZIP, extract `gog.exe`, and add the
directory to `PATH`.

## Docker

Release tags publish a GitHub Container Registry image:

```bash
docker run --rm ghcr.io/steipete/gogcli:latest version
docker run --rm ghcr.io/steipete/gogcli:v0.15.0 version
```

Authenticated container runs should mount a persistent config directory and use
the encrypted file keyring:

```bash
docker volume create gogcli-config

docker run --rm -it \
  -e GOG_KEYRING_BACKEND=file \
  -e GOG_KEYRING_PASSWORD \
  -v gogcli-config:/home/gog/.config/gogcli \
  ghcr.io/steipete/gogcli:latest \
  auth add you@gmail.com --services gmail,calendar,drive
```

Keep `GOG_KEYRING_PASSWORD` in the shell session or CI secret store. Do not bake
it into images, scripts, or checked-in profiles.

## Source Builds

```bash
git clone https://github.com/steipete/gogcli.git
cd gogcli
make
./bin/gog --version
```

Source builds require the Go version declared in `go.mod`.

## Related Command Pages

- [`gog version`](commands/gog-version.md)
- [`gog auth keyring`](commands/gog-auth-keyring.md)
- [`gog auth credentials`](commands/gog-auth-credentials.md)
- [`gog auth add`](commands/gog-auth-add.md)
