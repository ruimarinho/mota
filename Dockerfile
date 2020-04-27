FROM scratch
COPY shelly-updater /
ENTRYPOINT ["/shelly-updater"]
