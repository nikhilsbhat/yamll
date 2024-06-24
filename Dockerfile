### Description: Dockerfile for yamll
FROM alpine:3.16

COPY yamll /

# Starting
ENTRYPOINT [ "/yamll" ]