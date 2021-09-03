FROM fedora:34
ENV TASK_NAME=import-ami

COPY ${TASK_NAME} /usr/local/bin/${TASK_NAME} 
COPY entrypoint /usr/local/bin/entrypoint
COPY user_setup /usr/local/bin/user_setup

ENV ENTRY_CMD=/usr/local/bin/${TASK_NAME} \
    USER_UID=1001 \
    USER_NAME=${TASK_NAME}

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

