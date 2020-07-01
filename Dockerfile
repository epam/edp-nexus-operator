FROM alpine:3.11.6

ENV OPERATOR=/usr/local/bin/nexus-operator \
    USER_UID=1001 \
    USER_NAME=nexus-operator \
    HOME=/home/nexus-operator

# install operator binary
COPY nexus-operator ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/configs /usr/local/configs

RUN chmod u+x /usr/local/bin/user_setup && \
    chmod ugo+x /usr/local/bin/entrypoint && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}