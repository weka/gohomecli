⚠️ This repository is deprecated. The CLI has been moved to weka/gohome.

# Weka Home Command Line Utility

### Configuration

#### Initial configuration file create

- Run `gohomecli config site list`, on first run it will auto-create configuration file, **without** setting API key
- Set api key: edit `~/.config/home-cli/config.toml`. API Key retrieved from `https://home.weka.io/user-management` or similar URL from different deployment

#### Adding more sites

`homecli config site add <site> <cloud-url> <api-key>`

For example, to add new localhost deployment:

- `homecli config site add localhost http://localhost:8000 API_KEY_FOR_LOCALHOST`

#### Setting default site

- `gohomecli config default-site localhost` (`localhost` is a name used during `site add`)

#### Direct config file editing

Config file can be edited directly, it should look like:

```
default_site = "prod"

[sites]

  [sites.prod]
    api_key = "key1"
    cloud_url = "https://api.home.weka.io/"

  [sites.another]
    api_key = "key"
    cloud_url = "https://api.another.deployment"


  [sites.local]
    api_key = "key3"
    cloud_url = "http://localhost:8000"
```

## Using different site

Every command has `--site` flag to point to specific site, using name that was added during create

## Prepare new release

### Create tag

To prepare new release, first of all need to create new tag

```shell
# git tag v<TAG_MAJOR>.<TAG_MINOR>.<TAG_PATCH> -v '<comment>'
git tag v0.4.14 -v 'added command homecli local status'

git push origin v0.4.14
```

### Build homecli binary

To generate `homecli` binary in root of repository run

```shell
./build.sh
```

For mac users there will be error message `./build.sh: line 20: upx-4.2.2-amd64_linux/upx: cannot execute binary file`. Ignore it.

As a result, there should be to files in **_bin_** directory:

```shell
bin
├── homecli_darwin_amd64
└── homecli_linux_amd64
```

### Create release

1. Go to [releases](https://github.com/weka/gohomecli/releases "releases")
2. click `Draft a new release` button
3. `Choose a tag` - select created tag
4. `Generate release notes`
5. `Attach binaries by dropping them here or selecting them` - select **homecli\_\*\_amd64** files from **_bin_** directory
6. `Publish release`
