# Multi-stage build: compile the React app, then serve the static output
# with nginx. Vite env vars (VITE_CATALOG_URL etc.) are baked in at build
# time, not read at container runtime — the defaults in src/api.ts already
# point at localhost:8081/8082/8083, which is correct here because the
# browser itself runs on the host, not inside the Docker network, so it
# reaches the backend via the host-mapped ports in docker-compose.yml either
# way.
FROM node:22-alpine AS build
WORKDIR /src
COPY package*.json ./
RUN npm install --registry https://registry.npmjs.org
COPY . .
ARG VITE_CLOUDBEES_SDK_KEY
ENV VITE_CLOUDBEES_SDK_KEY=$VITE_CLOUDBEES_SDK_KEY
RUN npm run build

FROM nginx:alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
