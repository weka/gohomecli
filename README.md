> [!CAUTION]
> **This repository is deprecated and no longer builds.**
>
> The Weka Home CLI has moved to **[weka/gohome](https://github.com/weka/gohome)**.
> Every build entry point here (`make`, `./build.sh`, `./deploy.sh`, `go build`,
> the release workflow) fails on purpose so stale clones stop shipping from here.
>
> ```
> git clone git@github.com:weka/gohome.git
> ```
>
> If you have local work in this clone, re-apply it against `weka/gohome`.
> The documentation below is kept for historical reference only.

# Weka Home Command Line Utility


### Configuration

#### Initial configuration file create
- Run `gohomecli config site list`, on first run it will auto-create configuration file, **without** setting API key
- Set api key: edit `~/.config/home-cli/config.toml`. API Key retrieved from `https://home.weka.io/api-keys` or similar URL from different deployment

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
