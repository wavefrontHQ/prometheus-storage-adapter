FROM       harbor-repo.vmware.com/dockerhub-proxy-cache/library/busybox:1.29.2-glibc

COPY target/adapter_linux /bin/adapter

USER        nobody
ENTRYPOINT  [ "/bin/adapter" ]
