# home-dns
Fast DNS for the Home

## About:

home-dns makes it easy to setup your own super fast DNS forwarder with caching, prefetching, and stale answer serving for worst case scenarios. 

## How to Use:

Just run the super simple install script:

```
sudo ./install
```

The script will show you every command it is running (which isn't many) and then start the systemd service for you. Everytime you reboot your machine, the dns server will start automatically.
