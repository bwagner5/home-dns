# home-dns
Fast DNS for the Home

## About:

home-dns makes it easy to setup your own super fast DNS forwarder with caching, prefetching, and stale answer serving for worst case scenarios. 

## How to Run:

docker run -p 5353:53/tcp -p 5353:53/udp -d homedns
