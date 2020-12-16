FROM node:12 as dev
WORKDIR /usr/src/app
COPY package*.json ./
RUN yarn install
COPY . .
RUN yarn run build

FROM bitnami/nginx as prod
COPY --from=dev /usr/src/app/dist /app