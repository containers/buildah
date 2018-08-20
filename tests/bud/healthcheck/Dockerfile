FROM alpine
HEALTHCHECK --start-period=10m --interval=5m --timeout=3s --retries=4 \
  CMD curl -f http://localhost/ || exit 1
