### Description: Dockerfile for yamll
FROM alpine:3.20.2

COPY yamll /

# Starting
ENTRYPOINT [ "/yamll" ]