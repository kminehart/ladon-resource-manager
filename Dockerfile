FROM scratch
ADD ./bin /
CMD ["/ladon-resource-manager", "-v=3", "-logtostderr=true"]
