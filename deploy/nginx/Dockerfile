# Reverse proxy in front of the frontend and backend services. The official
# image renders /etc/nginx/templates/*.template into conf.d at startup,
# substituting defined environment variables (NGINX_PORT) only.
FROM nginx:1.30-alpine
ENV NGINX_PORT=8088
COPY templates/ /etc/nginx/templates/
