FROM alpine:3.16
ARG VERSION
ARG GITHUB_REPOSITORY
WORKDIR /opt
USER root
ENV DOWNLOAD_LINK="https://github.com/${GITHUB_REPOSITORY}/releases/download/${VERSION}/otc-cli_${VERSION}_linux_amd64.tar.gz"
RUN echo $DOWNLOAD_LINK && \
    curl -LO $DOWNLOAD_LINK && \
    tar -zxvf helm-charts-manager* && \
    mv helm-charts-manager* /usr/local/bin/
