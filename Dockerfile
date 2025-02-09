FROM gcr.io/distroless/static-debian12
COPY cachembed /
ENTRYPOINT [ "/cachembed" ]
