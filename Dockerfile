FROM scratch

COPY lambo /lambo

EXPOSE 3000

ENTRYPOINT ["/lambo"]

