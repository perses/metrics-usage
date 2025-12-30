FROM alpine AS build-env
RUN apk add --update --no-cache mailcap

FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
LABEL maintainer="The Perses Authors <perses-team@googlegroups.com>"

COPY  ${TARGETPLATFORM}/metrics-usage                      /bin/metrics-usage
COPY  LICENSE                                              /LICENSE
COPY --from=build-env                                      /etc/mime.types /etc/mime.types

EXPOSE     8080
ENTRYPOINT [ "/bin/metrics-usage" ]
