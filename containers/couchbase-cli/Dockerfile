FROM python:2.7
RUN git clone https://github.com/couchbase/couchbase-cli.git
ENV CB_REST_USERNAME=Administrator \
     CB_REST_PASSWORD=password
WORKDIR couchbase-cli
ADD couchbase-cli-secure /couchbase-cli-secure
ENTRYPOINT ["./couchbase-cli"]
