### Description: Dockerfile for yamll
FROM alpine:3.23

ARG TARGETARCH

COPY dist/yamll_linux_${TARGETARCH}_*/yamll /

# Starting
ENTRYPOINT [ "/yamll" ]
