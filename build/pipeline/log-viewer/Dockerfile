# syntax=docker/dockerfile:1.7

# Tiny image: download a build log from the webhook host once at startup, then
# serve it through Monaco Editor (the engine behind VS Code) in read-only
# mode. Monaco is baked into the image so the viewer works without internet.

ARG MONACO_VERSION=0.52.2
ARG NODE_VERSION=24
ARG NGINX_VERSION=1.27

# Stage 1 — pull Monaco's prebuilt AMD distribution from npm. This stage is
# discarded; only the `min/vs/` tree is copied into the final image.
FROM node:${NODE_VERSION}-alpine AS monaco
ARG MONACO_VERSION
WORKDIR /tmp
RUN npm install --no-save --no-package-lock monaco-editor@${MONACO_VERSION}

# Stage 2 — runtime. Nginx + curl + Monaco bundle + tiny index.html.
FROM nginx:${NGINX_VERSION}-alpine

RUN apk add --no-cache curl ca-certificates

COPY --from=monaco /tmp/node_modules/monaco-editor/min/vs /usr/share/nginx/html/monaco/vs

RUN mkdir -p /usr/share/nginx/html/themes
COPY index.html      /usr/share/nginx/html/index.html
COPY NightOwl.json   /usr/share/nginx/html/themes/NightOwl.json
COPY nginx.conf      /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
