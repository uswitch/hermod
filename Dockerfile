FROM scratch

ADD bin/hermod-linux-amd64 hermod

ENTRYPOINT ["/hermod"]
