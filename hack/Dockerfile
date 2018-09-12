FROM fedora:latest
RUN dnf -y update; dnf -y clean all
RUN dnf -y install nginx --setopt install_weak_deps=false; dnf -y clean all
RUN echo "daemon off;" >> /etc/nginx/nginx.conf
RUN echo "nginx on Fedora" > /usr/share/nginx/html/index.html
EXPOSE 80
CMD [ "/usr/sbin/nginx" ]
