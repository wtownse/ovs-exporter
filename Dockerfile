FROM centos:7
ENV PORT=${PORT:-:9700}
COPY /ovs_info /ovs_info
ADD ovs_info.sh /ovs_info.sh
RUN chmod +x ovs_info && chmod +x ovs_info.sh \
&& yum install -y docker-client
ENTRYPOINT ["/ovs_info.sh"]
EXPOSE $PORT
