FROM centos:7

RUN yum update -y && yum install -y unbound

ADD home-dns-unbound.conf /etc/unbound/unbound.conf

ENTRYPOINT ["/usr/sbin/unbound"]
