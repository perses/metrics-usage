FROM alpine AS build-env
RUN apk add --update --no-cache mailcap

FROM gcr.io/distroless/static-debian11

LABEL maintainer="The Perses Authors <perses-team@googlegroups.com>"

USER nobody

COPY --chown=nobody:nobody metrics-usage                      /bin/metrics-usage
COPY --chown=nobody:nobody LICENSE                           /LICENSE
COPY --from=build-env --chown=nobody:nobody                  /etc/mime.types /etc/mime.types

WORKDIR /perses

EXPOSE     8080
ENTRYPOINT [ "/bin/metrics-usage" ]
