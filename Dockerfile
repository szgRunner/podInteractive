FROM www.registry.it/busybox
COPY app .
COPY view ./view/
COPY static ./static/
RUN chmod +x app
CMD ["./app","--kubeConfig="]