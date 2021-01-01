FROM centos:7

RUN yum update -y && yum install -y unbound

RUN echo "server:" > /etc/unbound/blacklist.conf && curl -s https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts | grep '^0\.0\.0\.0 \.*' | cut -d' ' -f2 | sed 's/\(.*\)/ local-data: "\1 A 0.0.0.0"/' >> /etc/unbound/blacklist.conf 

ADD home-dns-unbound.conf /etc/unbound/unbound.conf

ENTRYPOINT ["/usr/sbin/unbound"]
