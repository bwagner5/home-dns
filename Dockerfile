FROM ubuntu:18.04

RUN apt-get update -y && apt-get install -y unbound

ADD home-dns-unbound.conf /etc/unbound/unbound.conf

ENTRYPOINT ["/usr/sbin/unbound", "-d"]
