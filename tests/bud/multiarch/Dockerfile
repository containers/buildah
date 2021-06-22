FROM alpine AS base
RUN cp /etc/apk/arch /root/arch-base

FROM alpine
# Make sure that non-default arch doesn't mess with copying from previous stages.
COPY --from=base /root/arch-base /root/
# Make sure that COPY --from=image uses the image for the preferred architecture.
COPY --from=alpine /etc/apk/arch /root/
RUN cmp /etc/apk/arch /root/arch
RUN cmp /etc/apk/arch /root/arch-base
