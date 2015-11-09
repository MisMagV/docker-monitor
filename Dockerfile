FROM scratch
MAINTAINER YI-HUNG JEN <yihungjen@gmail.com>

COPY ca-certificates.crt /etc/ssl/certs/
COPY docker-monitor /
ENTRYPOINT ["/docker-monitor"]
CMD ["--help"]

EXPOSE 9091
