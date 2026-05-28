# Build backend binary file
FROM golang:1.25.5-alpine3.23 AS be-builder
ARG RELEASE_BUILD
ARG BUILD_PIPELINE
ARG BUILD_UNIXTIME
ARG BUILD_DATE
ARG CHECK_3RD_API
ARG SKIP_TESTS
ARG RUN_BUILD_CHECKS=0
ENV RELEASE_BUILD=$RELEASE_BUILD
ENV BUILD_PIPELINE=$BUILD_PIPELINE
ENV BUILD_UNIXTIME=$BUILD_UNIXTIME
ENV BUILD_DATE=$BUILD_DATE
ENV CHECK_3RD_API=$CHECK_3RD_API
ENV SKIP_TESTS=$SKIP_TESTS
ENV RUN_BUILD_CHECKS=$RUN_BUILD_CHECKS
WORKDIR /go/src/github.com/mayswind/financially_free
COPY . .
RUN sed -i 's/\r$//' build.sh docker/backend-build-pre-setup.sh
RUN sh docker/backend-build-pre-setup.sh
RUN apk add git gcc g++ libc-dev
RUN if [ "$RUN_BUILD_CHECKS" = "1" ]; then sh ./build.sh backend; else sh ./build.sh backend --no-lint --no-test; fi

# Build frontend files
FROM --platform=$BUILDPLATFORM node:24.12.0-alpine3.23 AS fe-builder
ARG RELEASE_BUILD
ARG BUILD_PIPELINE
ARG BUILD_UNIXTIME
ARG BUILD_DATE
ARG RUN_BUILD_CHECKS=0
ENV RELEASE_BUILD=$RELEASE_BUILD
ENV BUILD_PIPELINE=$BUILD_PIPELINE
ENV BUILD_UNIXTIME=$BUILD_UNIXTIME
ENV BUILD_DATE=$BUILD_DATE
ENV RUN_BUILD_CHECKS=$RUN_BUILD_CHECKS
WORKDIR /go/src/github.com/mayswind/financially_free
COPY . .
RUN sed -i 's/\r$//' build.sh docker/frontend-build-pre-setup.sh
RUN sh docker/frontend-build-pre-setup.sh
RUN apk add git
RUN if [ "$RUN_BUILD_CHECKS" = "1" ]; then sh ./build.sh frontend; else sh ./build.sh frontend --no-lint --no-test; fi

# Package docker image
FROM alpine:3.23.2
LABEL maintainer="MaysWind <i@mayswind.net>"
RUN addgroup -S -g 1000 financially_free && adduser -S -G financially_free -u 1000 financially_free
RUN apk --no-cache add tzdata
COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
RUN sed -i 's/\r$//' /docker-entrypoint.sh && chmod +x /docker-entrypoint.sh
RUN mkdir -p /financially_free && chown 1000:1000 /financially_free \
  && mkdir -p /financially_free/data && chown 1000:1000 /financially_free/data \
  && mkdir -p /financially_free/log && chown 1000:1000 /financially_free/log \
  && mkdir -p /financially_free/storage && chown 1000:1000 /financially_free/storage
WORKDIR /financially_free
COPY --from=be-builder --chown=1000:1000 /go/src/github.com/mayswind/financially_free/financially_free /financially_free/financially_free
COPY --from=fe-builder --chown=1000:1000 /go/src/github.com/mayswind/financially_free/dist /financially_free/public
COPY --chown=1000:1000 conf /financially_free/conf
COPY --chown=1000:1000 templates /financially_free/templates
COPY --chown=1000:1000 LICENSE /financially_free/LICENSE
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
