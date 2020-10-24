FROM docker.io/panubo/sshd:1.3.0

RUN sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && mkdir -p /etc/entrypoint.d \
    && echo '[[ "${SSH_ENABLE_ROOT}" == "true" ]] && echo root:${SSH_ROOT_PASSWORD:-"root"} | chpasswd' > /etc/entrypoint.d/chpasswdroot \
    && chmod +x /etc/entrypoint.d/chpasswdroot