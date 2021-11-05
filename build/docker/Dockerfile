FROM docker.io/panubo/sshd:1.3.0

ENV SSH_ROOT_PASSWORD "root"
ENV SSH_ROOT_AUTHORIZED_KEY ""
RUN sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && sed -i 's/AllowTcpForwarding no/AllowTcpForwarding yes/' /etc/ssh/sshd_config \ 
    && mkdir -p /etc/entrypoint.d \
    && echo '[[ "${SSH_ENABLE_ROOT}" == "true" ]] && echo root:${SSH_ROOT_PASSWORD} | chpasswd' > /etc/entrypoint.d/root-chpasswd.sh \
    && echo 'echo -n "$SSH_ROOT_AUTHORIZED_KEY" >> /etc/authorized_keys/root && chmod 600 /etc/authorized_keys/root' > /etc/entrypoint.d/root-authorized-key.sh \
    && chmod +x /etc/entrypoint.d/*