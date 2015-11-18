FROM scratch
MAINTAINER YI-HUNG JEN <yihungjen@gmail.com>

COPY ca-certificates.crt /etc/ssl/certs/
COPY docker-monitor /
ENTRYPOINT ["/docker-monitor"]
CMD ["--help"]

ENV NODE_NAME ""
ENV NODE_AVAIL_ZONE ""
ENV NODE_REGION ""
ENV NODE_PUBLIC_IPV4 ""
ENV NODE_PRIVATE_IPV4 ""
